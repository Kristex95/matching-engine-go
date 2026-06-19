package orderbook

import "fmt"

const (
	Reset  = "\033[0m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Cyan   = "\033[36m"
)

func (ob *OrderBook) Print() {
	fmt.Printf("\n\n\n%s====== %s ======%s", Cyan, ob.BaseCurrency+"/"+ob.QuoteCurrency, Reset)
    fmt.Printf("\n%s======== ASK ========%s\n", Red, Reset)
    for price, level := range ob.Asks {
        fmt.Printf("%sPrice: %f%s\n", Yellow, price, Reset) 
        for _, o := range level.Orders {
            fmt.Printf("  OrderID=%s Side=%s Price=%f Amount=%f\n",
                o.ID, o.Side, o.Price, o.Amount)
        }
    }
    fmt.Printf("%s=====================%s\n\n", Red, Reset)

    fmt.Printf("%s======== BID ========%s\n", Green, Reset)
    for price, level := range ob.Bids {
        fmt.Printf("%sPrice: %f%s\n", Cyan, price, Reset)
        for _, o := range level.Orders {
            fmt.Printf("  OrderID=%s Side=%s Price=%f Amount=%f\n",
                o.ID, o.Side, o.Price, o.Amount)
        }
    }
    fmt.Printf("%s=====================%s\n\n\n", Green, Reset)
}