# Config v1
Config defines the v1 version structure of the gitpod config file

## Supported parameters
| Property | Required | Description | Notes |
| --- | --- | --- | --- |
| `kind` | Y | Installation type to run |  |
| `domain` | Y | The domain to deploy to |  |
| `metadata` | N |  |  |
| `repository` | Y |  |  |
| `observability` | N |  |  |
| `analytics` | N |  |  |
| `database` | Y |  |  |
| `objectStorage` | Y |  |  |
| `containerRegistry` | Y |  |  |
| `certificate` | Y |  |  |
| `imagePullSecrets` | N |  |  |
| `workspace` | Y |  |  |
| `openVSX` | N |  |  |
| `authProviders` | N |  |  |
| `blockNewUsers` | N |  |  |
| `license` | N |  |  |
| `sshGatewayHostKey` | N |  |  |
| `disableDefinitelyGp` | N |  |  |
| `experimental` | N |  |  |
| `apiVersion` | Y | API version of the Gitpod config defintion. `v1` in this version of Config |  |
