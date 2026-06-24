package orderbook

type Snapshot struct {
    Symbol string      `json:"symbol"`
    Bids   [][]string `json:"bids"`
    Asks   [][]string `json:"asks"`
}