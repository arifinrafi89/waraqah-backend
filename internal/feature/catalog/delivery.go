package catalog

// DeliveryArea is where an order ships to, as far as delivery time is concerned.
type DeliveryArea string

// The two areas (delivery_area.dart).
const (
	InsideDhaka  DeliveryArea = "insideDhaka"
	OutsideDhaka DeliveryArea = "outsideDhaka"
)

// EstimateKind says how an Edition reaches the reader.
type EstimateKind string

// The estimate kinds, as the Dart sealed classes.
const (
	InstantDownload EstimateKind = "instantDownload"
	ShipsOnRelease  EstimateKind = "shipsOnRelease"
	Unavailable     EstimateKind = "unavailable"
	ShipsInDays     EstimateKind = "shipsInDays"
)

// Estimate is DeliveryEstimate: when an Edition would reach the reader.
type Estimate struct {
	Kind    EstimateKind
	MinDays int
	MaxDays int
}

// PrintedEstimate is how long a printed book in stock takes to reach an area.
func PrintedEstimate(area DeliveryArea) Estimate {
	if area == InsideDhaka {
		return Estimate{Kind: ShipsInDays, MinDays: 1, MaxDays: 2}
	}
	return Estimate{Kind: ShipsInDays, MinDays: 3, MaxDays: 5}
}

// DeliveryEstimateOf is DeliveryEstimate.of:
//   - eBooks download straight away
//   - printed books in stock take 1 to 2 days inside Dhaka, 3 to 5 outside
//   - pre-orders ship on release
//   - out-of-stock editions can not be delivered
func DeliveryEstimateOf(e Edition, area DeliveryArea) Estimate {
	switch {
	case e.Format == "ebook":
		return Estimate{Kind: InstantDownload}
	case e.Stock > 0:
		return PrintedEstimate(area)
	case e.IsPreorder:
		return Estimate{Kind: ShipsOnRelease}
	}
	return Estimate{Kind: Unavailable}
}
