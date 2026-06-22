package orderbook

type Snapshot struct {
    Symbol string      `json:"symbol"`
    Bids   [][]float64 `json:"bids"`
    Asks   [][]float64 `json:"asks"`
}