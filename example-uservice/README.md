# Tat uService Example

uService example, with :

* Initialize Tat Client
* Requesting Tat Engine


## Build

```bash
cd api
go build
./api -h
```

Run with use environment variable instead of argd parameter.
```bash
export TAT_USERVICE_URL_TAT_ENGINE="http://url-tat-egine"
export TAT_USERVICE_USERNAME_TAT_ENGINE="your-tat-username"
export TAT_USERVICE_PASSWORD_TAT_ENGINE="your-tat-password"

# then run uservice
./api
```
