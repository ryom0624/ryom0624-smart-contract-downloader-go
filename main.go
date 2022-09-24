package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Env struct {
	EtherscanAPIKey string

	// todo
	//BSCScanAPIKey     string
	//PolygonscanAPIKey string
}

func loadEnv() Env {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatalln(err)
	}
	return Env{
		EtherscanAPIKey: os.Getenv("ETHERSCAN_APIKEY"),
		//BSCScanAPIKey:     os.Getenv("BSCSCAN_APIKEY"),
		//PolygonscanAPIKey: os.Getenv("POLYGONSCAN_APIKEY"),
	}
}

var OutputDir = "./output"
var SourceKey = "content"

func main() {
	log.Println("Running SmartContract Downloader. Only Ethereum Main-net.")

	address := flag.String("address", "", "enter you want to get smart contract address")
	flag.Parse()

	if *address == "" {
		log.Println("↓ input smart contract address ↓")
		var input string
		fmt.Scanln(&input)

		*address = input
	}

	log.Println("ContractAddress is", *address)

	if !strings.Contains(*address, "0x") {
		log.Println("invalid ethereum address")
		os.Exit(1)
	}

	env := loadEnv()

	if env.EtherscanAPIKey == "" {
		log.Println("expected scan api key")
		os.Exit(1)
	}

	log.Println("getting...")

	fetchResult, err := contractFetcher(address, env.EtherscanAPIKey)
	if err != nil {
		log.Printf("contractFetcher err: %s", err)
		os.Exit(1)
	}

	contractName := fetchResult.ContractName
	log.Println("contract name is: ", contractName)

	parsedContract, err := parseContract(fetchResult.SourceCode, contractName)
	if err != nil {
		log.Printf("parseContract err: %s", err)
		os.Exit(1)
	}

	if err := writeContractsToZip(contractName, *address, parsedContract); err != nil {
		log.Printf("writeContractsToZip err: %s", err)
		os.Exit(1)
	}

	log.Printf("Success to download contractname=%s address=%s", contractName, *address)
}

type Contracts map[string]map[string]string

type ScanResponse struct {
	Status  string       `json:"status"`
	Message string       `json:"message"`
	Result  []ScanResult `json:"result"`
}

type ScanResult struct {
	SourceCode           string `json:"SourceCode"`
	ABI                  string `json:"ABI"`
	ContractName         string `json:"ContractName"`
	CompilerVersion      string `json:"CompilerVersion"`
	OptimizationUsed     string `json:"OptimizationUsed"`
	Runs                 string `json:"Runs"`
	ConstructorArguments string `json:"ConstructorArguments"`
	EVMVersion           string `json:"EVMVersion"`
	Library              string `json:"Library"`
	LicenseType          string `json:"LicenseType"`
	Proxy                string `json:"Proxy"`
	Implementation       string `json:"Implementation"`
	SwarmSource          string `json:"SwarmSource"`
}

type StandardJsonInputFormat struct {
	Language string           `json:"language"`
	Sources  *json.RawMessage `json:"sources"`
	Settings struct {
		Remappings []interface{} `json:"remappings"`
		Optimizer  struct {
			Enabled bool `json:"enabled"`
			Runs    int  `json:"runs"`
		} `json:"optimizer"`
		EvmVersion string `json:"evmVersion"`
		Libraries  struct {
		} `json:"libraries"`
		OutputSelection struct {
			Field1 struct {
				Field1 []string `json:"*"`
			} `json:"*"`
		} `json:"outputSelection"`
	} `json:"settings"`
}

func contractFetcher(address *string, apiKey string) (*ScanResult, error) {
	resp, err := http.Get(fmt.Sprintf("https://api.etherscan.io/api?module=contract&action=getsourcecode&address=%s&apikey=%s", *address, apiKey))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var fetchScanResponse ScanResponse
	if err := json.NewDecoder(resp.Body).Decode(&fetchScanResponse); err != nil {
		return nil, err
	}

	if fetchScanResponse.Message != "OK" {
		return nil, errors.New("failed to fetch " + fetchScanResponse.Message)
	}

	if len(fetchScanResponse.Result) != 1 {
		return nil, errors.New("unexpected result")
	}

	if fetchScanResponse.Result[0].ABI == "Contract source code not verified" {
		return nil, errors.New("not verified contract")
	}

	if fetchScanResponse.Result[0].Proxy == "1" && strings.ToLower(fetchScanResponse.Result[0].Implementation) != strings.ToLower(*address) {
		log.Printf("get proxy address( %s ). refetching implementation address: %s", address, fetchScanResponse.Result[0].Implementation)
		*address = fetchScanResponse.Result[0].Implementation
		return contractFetcher(address, apiKey)
	}

	return &fetchScanResponse.Result[0], nil
}

/*
	・Solidity Multiple files format（path -> contract type）
	・Solidity Standard Json-Input format（double curly braces）
	・Single File Format
*/
func parseContract(source, contractName string) (*Contracts, error) {

	// SolidityMultipleFilesFormat
	var parsedContract Contracts
	if err := parseMultipleFilesFormatContract(source, &parsedContract); err != nil {

		if strings.HasPrefix(source, "{{") {
			// SolidityStandardJsonInputFormat
			return parseStandardJsonInputFormatContract(source)
		} else {
			// SingleFile
			return parseSingleFileFormatContract(source, contractName)
		}
	}

	return &parsedContract, nil
}

func parseMultipleFilesFormatContract(source string, contracts *Contracts) error {
	return json.Unmarshal([]byte(source), &contracts)
}

func parseSingleFileFormatContract(source, contractName string) (*Contracts, error) {
	var parsedContract Contracts
	parsedContract = map[string]map[string]string{
		fmt.Sprintf("%s/Contract.sol", contractName): {SourceKey: source},
	}
	return &parsedContract, nil
}

func parseStandardJsonInputFormatContract(source string) (*Contracts, error) {
	// trim a curly brace {}
	trimmedSourceCode := source[1 : len(source)-1]

	var parsedSourceCode StandardJsonInputFormat
	if err := json.Unmarshal([]byte(trimmedSourceCode), &parsedSourceCode); err != nil {
		return nil, fmt.Errorf("%w - %s", err, trimmedSourceCode)
	}
	var parsedContract Contracts
	if err := json.Unmarshal(*parsedSourceCode.Sources, &parsedContract); err != nil {
		return nil, err
	}
	return &parsedContract, nil
}

func writeContractsToZip(contractName, address string, contracts *Contracts) error {
	archive, err := os.Create(fmt.Sprintf("%s/%s_%s.zip", OutputDir, contractName, address))
	if err != nil {
		return err
	}
	defer archive.Close()

	zipWriter := zip.NewWriter(archive)
	defer zipWriter.Close()

	for filename, source := range *contracts {
		w, err := zipWriter.Create(fmt.Sprintf("%s", filename))
		if err != nil {
			return err
		}

		if _, err := io.Copy(w, bytes.NewBufferString(source[SourceKey])); err != nil {
			return err
		}
	}

	return nil
}
