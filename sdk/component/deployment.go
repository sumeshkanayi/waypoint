package component

// DeploymentConfig is the configuration for the behavior of a deployment.
// Platforms should take this argument and use the value to set the appropriate
// settings for the deployment
type DeploymentConfig struct {
	Id             string
	ServerAddr     string
	ServerInsecure bool
}

// Env returns the environment variables that should be set for the entrypoint
// binary to have the proper configuration.
func (c *DeploymentConfig) Env() map[string]string {
	results := map[string]string{
		"DEVFLOW_DEPLOYMENT_ID": c.Id,
	}

	if c.ServerAddr == "" {
		// If the server is disabled we set this env var. Note that having
		// no address given also causes it to behave the same way.
		results["DEVFLOW_SERVER_DISABLE"] = "1"
	} else {
		// Note the server address.
		results["DEVFLOW_SERVER_ADDR"] = c.ServerAddr
		if c.ServerInsecure {
			results["DEVFLOW_SERVER_INSECURE"] = "1"
		}
	}

	return results
}