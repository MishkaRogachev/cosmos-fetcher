# cosmos-fetcher

![Go CI](https://github.com/MishkaRogachev/cosmos-fetcher/actions/workflows/ci.yml/badge.svg)

### Flag Descriptions

- **`--node-url`**: The URL of the Cosmos node's RPC endpoint.
- **`--start-height`**: The starting block height to begin fetching from. The default is the earliest available height.
- **`--end-height`**: The ending block height to stop fetching at. The default is the latest available height.
- **`--parallelism`**: The number of parallel fetchers to use for fetching blocks concurrently. Default is 5.
- **`--file-blocks`**: The number of blocks to store per JSON file. Default is 16.
- **`--max-retries`**: The maximum number of retries allowed when fetching a block fails. Default is 3 retries.
- **`--retry-interval`**: The time interval between retries in milliseconds. Default is 500 milliseconds.
- **`--list-ranges`**: If set, the application will list available block ranges and exit without fetching.

## Examples

1. List the available block ranges and exit
```
go run cmd/main.go -list-ranges -node-url https://rpc.testcosmos.directory/cosmosicsprovidertestnet
```

2. Crop by earliest block height
```
go run cmd/main.go -start-height 4000000 -end-height 4285100 -node-url https://rpc.provider-sentry-01.ics-testnet.polypore.xyz
```

3. Crop by latest block height
```
go run cmd/main.go -start-height 9372600 -end-height 9999999 -node-url https://rpc.provider-sentry-01.ics-testnet.polypore.xyz
```

4. Configure number of workers
```
go run cmd/main.go -parallelism 20 -node-url https://rpc.provider-sentry-01.ics-testnet.polypore.xyz
```

5. Configure retries
```
go run cmd/main.go -max-retries 5 -retry-interval 100 -start-height 9449000 -end-height 9449500 -node-url https://rpc.testcosmos.directory/cosmosicsprovidertestnet
```

6. Configure blocks per file
```
go run cmd/main.go -file-blocks 100 -start-height 23253000 -end-height 23254000 -node-url https://cosmos-rpc.publicnode.com:443
```