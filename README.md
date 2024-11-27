# cosmos-fetcher




## Examples

1. List the available block ranges and exit
```
go run cmd/main.go -list-ranges  -node-url https://rpc.provider-sentry-01.ics-testnet.polypore.xyz
```

2. Crop by earliest block height
```
go run cmd/main.go -start-height 4000000 -end-height 4285100  -node-url https://rpc.provider-sentry-01.ics-testnet.polypore.xyz


3. Crop by latest block height
```
go run cmd/main.go -start-height 9372600 -end-height 9999999  -node-url https://rpc.provider-sentry-01.ics-testnet.polypore.xyz
```

4. Configure number of workers
```
go run cmd/main.go -parallelism 20 -node-url https://rpc.provider-sentry-01.ics-testnet.polypore.xyz
```
