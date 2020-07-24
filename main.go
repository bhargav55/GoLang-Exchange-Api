package main

import (
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	shrimpyclient "github.com/ashman1984/shrimpy-go"

	"github.com/gin-gonic/gin"
)

// Env struct for the methods to implement
type Env struct {
	sc *shrimpyclient.Client
}

// Asset model
type Asset struct {
	Name                string
	Symbol              string
	PriceUsd            string
	PriceBtc            string
	PercentChange24HUsd string
	LastUpdated         time.Time
}

// Response model
type Response struct {
	AssetDetails map[string]Asset
	Message      string
}

func main() {

	router := gin.Default()
	var config shrimpyclient.Config
	config.Endpoint = "https://dev-api.shrimpy.io"

	config.DebugMessages = false

	sc := shrimpyclient.NewClient(config)
	env := &Env{sc: sc}

	router.GET("/exchange-rate", env.getExchangeRate)

	if err := router.Run(":9090"); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}

func (e *Env) getExchangeRate(ctx *gin.Context) {

	params := make(map[string]string)
	params["fromCurrency"] = ctx.Query("fromCurrency")
	params["toCurrency"] = ctx.Query("toCurrency")
	params["exchange"] = ctx.Query("exchange")

	if params["fromCurrency"] == "" || params["toCurrency"] == "" || params["exchange"] == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"status": http.StatusBadRequest, "message": "invalid parameters. Please provide valid parameters"})
		return
	}

	response := getExchangeTickers(e, params)

	if response.AssetDetails == nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": response.Message})
		return
	}

	priceUsd1, _ := strconv.ParseFloat(response.AssetDetails[params["fromCurrency"]].PriceUsd, 64)

	priceUsd2, _ := strconv.ParseFloat(response.AssetDetails[params["toCurrency"]].PriceUsd, 64)
	result := priceUsd1 / priceUsd2
	ctx.JSON(http.StatusOK, gin.H{"status": http.StatusOK, "data": result})
	return
}

func getExchangeTickers(e *Env, params map[string]string) Response {

	// get the exchange rates older <58 sec from shrimpy
	for {
		getTickers := e.sc.GetExchangeTickers(params["exchange"])
		if getTickers == nil {

			return Response{Message: "Invalid exchange. This exchange is not supported"}
		}

		assetDetails := findAssets(getTickers, params["fromCurrency"], params["toCurrency"])

		if len(assetDetails) != 2 {
			return Response{Message: "Invalid Crypto Asset.  Please provide a valid crypto"}
		}

		diff1 := time.Now().UTC().Sub(assetDetails[params["fromCurrency"]].LastUpdated).Seconds()
		diff2 := time.Now().UTC().Sub(assetDetails[params["toCurrency"]].LastUpdated).Seconds()

		if diff1 < 58 || diff2 < 58 {
			return Response{AssetDetails: assetDetails}
		}

	}

}

// get crypto asset details from retrieved assets
func findAssets(slice shrimpyclient.Tickers, val1, val2 string) map[string]Asset {
	temp := make(map[string]Asset)
	count := 0
	for _, item := range slice {
		if strings.EqualFold(item.Symbol, val1) {
			temp[val1] = Asset{Name: item.Name, Symbol: item.Symbol, PriceUsd: item.PriceUsd, PriceBtc: item.PriceBtc, PercentChange24HUsd: item.PercentChange24HUsd, LastUpdated: item.LastUpdated}
			count++

		}
		if strings.EqualFold(item.Symbol, val2) {
			temp[val2] = Asset{Name: item.Name, Symbol: item.Symbol, PriceUsd: item.PriceUsd, PriceBtc: item.PriceBtc, PercentChange24HUsd: item.PercentChange24HUsd, LastUpdated: item.LastUpdated}
			count++
		}
		if count == 2 {
			return temp
		}

	}

	return nil
}
