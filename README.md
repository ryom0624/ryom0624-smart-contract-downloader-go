# Smart Contract Downloader in Go 

Now only Etherscan.

This is a smart contract downloader which uses Etherscan, PolygonScan(todo) and BSCScan(todo) API to get verified contracts source code.

After finding the desired contract you can download the package as a compressed zip file.

Inspired by [smart-contract-downloader](https://github.com/amimaro/smart-contract-downloader)

## Additional Feature

- If contract address is Proxy contract, refetch implementation address. 


# Getting Started

Copy .env.example to .env

    $ cp .env.example env

Need Etherscan account. If you don't have account, you can create [here](https://etherscan.io/register)


Set your Etherscan API Key at .env. 

See .env.example

    ETHERSCAN_APIKEY="your_api_key"

There are two patterns to set the contract address. 

You can set address with command line argument.

    $ go run main.go -address="enter_contract_address"

Others, no argument pattern.

    $ go run main.go
    > 2022/09/24 17:39:47 input address
    > //... enter here contract address



# Example

[UNI Token(ERC20)](https://etherscan.io/token/0x1f9840a85d5af5bf1d1762f925bdaddc4201f984)
        
    $ go run main.go -address="0x1f9840a85d5aF5bf1D1762F925BDADdC4201F984"
    2022/09/24 21:33:14 Running SmartContract Downloader. Only Ethereum Main-net.
    2022/09/24 21:33:14 ContractAddress is 0x1f9840a85d5aF5bf1D1762F925BDADdC4201F984
    2022/09/24 21:33:14 getting...
    2022/09/24 21:33:15 contract name is:  Uni
    2022/09/24 21:33:15 Success to download contractname=Uni address=0x1f9840a85d5aF5bf1D1762F925BDADdC4201F984


Default output dir is `./output/{contract_name}_{contract_address}.zip`.

    
## License

MIT [LICENSE.md](https://github.com/ryom0624/smart-contract-downloader-go/blob/master/LICENSE.md)

