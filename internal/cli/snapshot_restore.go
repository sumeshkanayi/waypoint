package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/hashicorp/waypoint/internal/pkg/flag"
	pb "github.com/hashicorp/waypoint/internal/server/gen"
	"github.com/posener/complete"
	sshterm "golang.org/x/crypto/ssh/terminal"
)

type SnapshotRestoreCommand struct {
	*baseCommand
}

// initWriter inspects args to figure out where the snapshot will be read from. It
// supports args[0] being '-' to force reading from stdin.
func (c *SnapshotRestoreCommand) initReader(args []string) (io.Reader, io.Closer, error) {
	if len(args) >= 1 {
		if args[0] == "-" {
			return os.Stdin, nil, nil
		}

		f, err := os.Open(args[0])
		if err != nil {
			return nil, nil, err
		}

		return f, f, nil
	}

	f := os.Stdin

	if sshterm.IsTerminal(int(f.Fd())) {
		return nil, nil, fmt.Errorf("stdin is a terminal, refusing to use (use '-' to force)")
	}

	return f, nil, nil
}

func (c *SnapshotRestoreCommand) Run(args []string) int {
	// Initialize. If we fail, we just exit since Init handles the UI.
	if err := c.Init(
		WithArgs(args),
		WithFlags(c.Flags()),
		WithNoConfig(),
	); err != nil {
		return 1
	}

	client := c.project.Client()

	r, closer, err := c.initReader(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open output: %s", err)
		return 1
	}

	if closer != nil {
		defer closer.Close()
	}

	stream, err := client.RestoreSnapshot(c.Ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to restore snapshot: %s", err)
		return 1
	}

	err = stream.Send(&pb.RestoreSnapshotRequest{
		Event: &pb.RestoreSnapshotRequest_Open_{
			Open: &pb.RestoreSnapshotRequest_Open{},
		},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to send start message: %s", err)
		return 1
	}

	// Write the data in smaller chunks so we don't overwhelm the grpc stream
	// processing machinary.
	var buf [1024]byte

	for {
		// use ReadFull here because if r is an OS pipe, each bare call to Read()
		// can result in just one or two bytes per call, so we want to batch those
		// up before sending them off for better performance.
		n, err := io.ReadFull(r, buf[:])
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			err = nil
		}

		if n == 0 {
			break
		}

		err = stream.Send(&pb.RestoreSnapshotRequest{
			Event: &pb.RestoreSnapshotRequest_Chunk{
				Chunk: buf[:n],
			},
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to write snapshot data: %s", err)
			return 1
		}
	}

	_, err = stream.CloseAndRecv()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to receive snapshot start message: %s", err)
		return 1
	}

	if r == os.Stdin {
		c.ui.Output("Server data restored.")
	} else {
		c.ui.Output("Server data restored from '%s'.", args[0])
	}

	return 0
}

func (c *SnapshotRestoreCommand) Flags() *flag.Sets {
	return c.flagSet(0, nil)
}

func (c *SnapshotRestoreCommand) AutocompleteArgs() complete.Predictor {
	return complete.PredictFiles("")
}

func (c *SnapshotRestoreCommand) AutocompleteFlags() complete.Flags {
	return c.Flags().Completions()
}

func (c *SnapshotRestoreCommand) Synopsis() string {
	return "Restore the state of the current server using a snapshot."
}

func (c *SnapshotRestoreCommand) Help() string {
	return formatHelp(`
Usage: waypoint server restore [<filenamp>]

	Restore the state of the current server using a snapshot.

	The argument should be to a file written previously by 'waypoint server snapshot'.
	If no name is specified and standard input is not a terminal, the backup will read from
	standard input. Using a name of '-' will force reading from standard input.
`)
}
