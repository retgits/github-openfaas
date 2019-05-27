# openfaas-githubissues

An [OpenFaaS](https://openfaas.com) function to query the [GitHub](https://github.com) API for new issues assigned to the current user.

## Deploy

### Secrets

To deploy this app to OpenFaaS you'll need to have a few secrets ready:

| Name               | Description                                    |
|--------------------|------------------------------------------------|
| github-accesstoken | The Personal Access Token to connect to GitHub |

### Template

This app makes use of a custom template: `faas-cli template pull https://github.com/retgits/of-templates`

## Sample message

To invoke this function, no additional data is required. Running `faas-cli invoke githubissues` will trigger the function.