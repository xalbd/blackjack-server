# Blackjack Server

## Configuration/Deploying

The following environment variables need to be provided:
| Name          | Source        |
| ------------- | ------------- |
| `FRONTEND`  | URL of frontend (refer to [xalbd/blackjack-app](https://github.com/xalbd/blackjack-app)) |

### Testing

Put the environment variable in a `.env` file in the repository root. Use `go run .` to launch the server on `localhost:8080`.
