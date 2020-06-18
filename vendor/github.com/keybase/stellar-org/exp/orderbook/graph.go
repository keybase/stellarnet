package orderbook

import (
	"sort"
	"sync"

	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

var (
	errOfferNotPresent     = errors.New("offer is not present in the order book graph")
	errEmptyOffers         = errors.New("offers is empty")
	errAssetAmountIsZero   = errors.New("current asset amount is 0")
	errSoldTooMuch         = errors.New("sold more than current balance")
	errBatchAlreadyApplied = errors.New("cannot apply batched updates more than once")
	errUnexpectedLedger    = errors.New("cannot apply unexpected ledger")
)

type sortByType string

const (
	sortBySourceAsset      sortByType = "source"
	sortByDestinationAsset sortByType = "destination"
)

// trading pair represents two assets that can be exchanged if an order is fulfilled
type tradingPair struct {
	// buyingAsset is obtained by calling offer.Buying.String() where offer is an xdr.OfferEntry
	buyingAsset string
	// sellingAsset is obtained by calling offer.Selling.String() where offer is an xdr.OfferEntry
	sellingAsset string
}

// OBGraph is an interface for orderbook graphs
type OBGraph interface {
	AddOffer(offer xdr.OfferEntry)
	Apply(ledger uint32) error
	Discard()
	Offers() []xdr.OfferEntry
	OffersMap() map[xdr.Int64]xdr.OfferEntry
	RemoveOffer(xdr.Int64) OBGraph
	Pending() ([]xdr.OfferEntry, []xdr.Int64)
	Clear()
}

// OrderBookGraph is an in memory graph representation of all the offers in the stellar ledger
type OrderBookGraph struct {
	// edgesForSellingAsset maps an asset to all offers which sell that asset
	// note that each key in the map is obtained by calling offer.Selling.String()
	// where offer is an xdr.OfferEntry
	edgesForSellingAsset map[string]edgeSet
	// edgesForBuyingAsset maps an asset to all offers which buy that asset
	// note that each key in the map is obtained by calling offer.Buying.String()
	// where offer is an xdr.OfferEntry
	edgesForBuyingAsset map[string]edgeSet
	// tradingPairForOffer maps an offer id to the assets which are being exchanged
	// in the given offer
	tradingPairForOffer map[xdr.Int64]tradingPair
	// batchedUpdates is internal batch of updates to this graph. Users can
	// create multiple batches using `Batch()` method but sometimes only one
	// batch is enough.
	// the orderbook graph is accurate up to lastLedger
	lastLedger     uint32
	batchedUpdates *orderBookBatchedUpdates
	lock           sync.RWMutex
}

var _ OBGraph = (*OrderBookGraph)(nil)

// NewOrderBookGraph constructs a new OrderBookGraph
func NewOrderBookGraph() *OrderBookGraph {
	graph := &OrderBookGraph{
		edgesForSellingAsset: map[string]edgeSet{},
		edgesForBuyingAsset:  map[string]edgeSet{},
		tradingPairForOffer:  map[xdr.Int64]tradingPair{},
	}

	graph.batchedUpdates = graph.batch()
	return graph
}

// AddOffer will queue an operation to add the given offer to the order book in
// the internal batch.
// You need to run Apply() to apply all enqueued operations.
func (graph *OrderBookGraph) AddOffer(offer xdr.OfferEntry) {
	graph.batchedUpdates.addOffer(offer)
}

// RemoveOffer will queue an operation to remove the given offer from the order book in
// the internal batch.
// You need to run Apply() to apply all enqueued operations.
func (graph *OrderBookGraph) RemoveOffer(offerID xdr.Int64) OBGraph {
	graph.batchedUpdates.removeOffer(offerID)
	return graph
}

// Pending returns a list of queued offers which will be added to the order book and
// a list of queued offers which will be removed from the order book.
func (graph *OrderBookGraph) Pending() ([]xdr.OfferEntry, []xdr.Int64) {
	var toUpdate []xdr.OfferEntry
	var toRemove []xdr.Int64
	for _, update := range graph.batchedUpdates.operations {
		if update.operationType == addOfferOperationType {
			toUpdate = append(toUpdate, *update.offer)
		} else if update.operationType == removeOfferOperationType {
			toRemove = append(toRemove, update.offerID)
		}
	}
	return toUpdate, toRemove
}

// Discard removes all operations which have been queued but not yet applied to the OrderBookGraph
func (graph *OrderBookGraph) Discard() {
	graph.batchedUpdates = graph.batch()
}

// Apply will attempt to apply all the updates in the internal batch to the order book.
// When Apply is successful, a new empty, instance of internal batch will be created.
func (graph *OrderBookGraph) Apply(ledger uint32) error {
	err := graph.batchedUpdates.apply(ledger)
	if err != nil {
		return err
	}
	graph.batchedUpdates = graph.batch()
	return nil
}

// Offers returns a list of offers contained in the order book
func (graph *OrderBookGraph) Offers() []xdr.OfferEntry {
	graph.lock.RLock()
	defer graph.lock.RUnlock()

	offers := []xdr.OfferEntry{}
	for _, edges := range graph.edgesForSellingAsset {
		for _, offersForEdge := range edges {
			offers = append(offers, offersForEdge...)
		}
	}

	return offers
}

// Clear removes all offers from the graph.
func (graph *OrderBookGraph) Clear() {
	graph.lock.Lock()
	defer graph.lock.Unlock()

	graph.edgesForSellingAsset = map[string]edgeSet{}
	graph.edgesForBuyingAsset = map[string]edgeSet{}
	graph.tradingPairForOffer = map[xdr.Int64]tradingPair{}
	graph.batchedUpdates = graph.batch()
	graph.lastLedger = 0
}

// OffersMap returns a ID => OfferEntry map of offers contained in the order
// book.
func (graph *OrderBookGraph) OffersMap() map[xdr.Int64]xdr.OfferEntry {
	offers := graph.Offers()
	m := make(map[xdr.Int64]xdr.OfferEntry, len(offers))

	for _, entry := range offers {
		m[entry.OfferId] = entry
	}

	return m
}

// Batch creates a new batch of order book updates which can be applied
// on this graph
func (graph *OrderBookGraph) batch() *orderBookBatchedUpdates {
	return &orderBookBatchedUpdates{
		operations: []orderBookOperation{},
		committed:  false,
		orderbook:  graph,
	}
}

// findOffers returns all offers for a given trading pair
// The offers will be sorted by price from cheapest to most expensive
// The returned offers will span at most `maxPriceLevels` price levels
func (graph *OrderBookGraph) findOffers(
	selling, buying string, maxPriceLevels int,
) []xdr.OfferEntry {
	results := []xdr.OfferEntry{}
	edges, ok := graph.edgesForSellingAsset[selling]
	if !ok {
		return results
	}
	offers, ok := edges[buying]
	if !ok {
		return results
	}

	for _, offer := range offers {
		// Offers are sorted by price, so, equal prices will always be contiguous.
		if len(results) == 0 || !results[len(results)-1].Price.Equal(offer.Price) {
			maxPriceLevels--
		}
		if maxPriceLevels < 0 {
			return results
		}

		results = append(results, offer)
	}
	return results
}

// FindAsksAndBids returns all asks and bids for a given trading pair
// Asks consists of all offers which sell `selling` in exchange for `buying` sorted by
// price (in terms of `buying`) from cheapest to most expensive
// Bids consists of all offers which sell `buying` in exchange for `selling` sorted by
// price (in terms of `selling`) from cheapest to most expensive
// Both Asks and Bids will span at most `maxPriceLevels` price levels
func (graph *OrderBookGraph) FindAsksAndBids(
	selling, buying xdr.Asset, maxPriceLevels int,
) ([]xdr.OfferEntry, []xdr.OfferEntry, uint32) {
	buyingString := buying.String()
	sellingString := selling.String()

	graph.lock.RLock()
	defer graph.lock.RUnlock()
	asks := graph.findOffers(sellingString, buyingString, maxPriceLevels)
	bids := graph.findOffers(buyingString, sellingString, maxPriceLevels)

	return asks, bids, graph.lastLedger
}

// add inserts a given offer into the order book graph
func (graph *OrderBookGraph) add(offer xdr.OfferEntry) error {
	if _, contains := graph.tradingPairForOffer[offer.OfferId]; contains {
		if err := graph.remove(offer.OfferId); err != nil {
			return errors.Wrap(err, "could not update offer in order book graph")
		}
	}

	sellingAsset := offer.Selling.String()
	buyingAsset := offer.Buying.String()
	graph.tradingPairForOffer[offer.OfferId] = tradingPair{
		buyingAsset:  buyingAsset,
		sellingAsset: sellingAsset,
	}
	if set, ok := graph.edgesForSellingAsset[sellingAsset]; !ok {
		graph.edgesForSellingAsset[sellingAsset] = edgeSet{}
		graph.edgesForSellingAsset[sellingAsset].add(buyingAsset, offer)
	} else {
		set.add(buyingAsset, offer)
	}

	if set, ok := graph.edgesForBuyingAsset[buyingAsset]; !ok {
		graph.edgesForBuyingAsset[buyingAsset] = edgeSet{}
		graph.edgesForBuyingAsset[buyingAsset].add(sellingAsset, offer)
	} else {
		set.add(sellingAsset, offer)
	}

	return nil
}

// remove deletes a given offer from the order book graph
func (graph *OrderBookGraph) remove(offerID xdr.Int64) error {
	pair, ok := graph.tradingPairForOffer[offerID]
	if !ok {
		return errOfferNotPresent
	}

	delete(graph.tradingPairForOffer, offerID)

	if set, ok := graph.edgesForSellingAsset[pair.sellingAsset]; !ok {
		return errOfferNotPresent
	} else if !set.remove(offerID, pair.buyingAsset) {
		return errOfferNotPresent
	} else if len(set) == 0 {
		delete(graph.edgesForSellingAsset, pair.sellingAsset)
	}

	if set, ok := graph.edgesForBuyingAsset[pair.buyingAsset]; !ok {
		return errOfferNotPresent
	} else if !set.remove(offerID, pair.sellingAsset) {
		return errOfferNotPresent
	} else if len(set) == 0 {
		delete(graph.edgesForBuyingAsset, pair.buyingAsset)
	}

	return nil
}

// IsEmpty returns true if the orderbook graph is not populated
func (graph *OrderBookGraph) IsEmpty() bool {
	graph.lock.RLock()
	defer graph.lock.RUnlock()

	return len(graph.edgesForSellingAsset) == 0
}

// FindPaths returns a list of payment paths originating from a source account
// and ending with a given destinaton asset and amount.
func (graph *OrderBookGraph) FindPaths(
	maxPathLength int,
	destinationAsset xdr.Asset,
	destinationAmount xdr.Int64,
	sourceAccountID *xdr.AccountId,
	sourceAssets []xdr.Asset,
	sourceAssetBalances []xdr.Int64,
	validateSourceBalance bool,
	maxAssetsPerPath int,
) ([]Path, uint32, error) {
	destinationAssetString := destinationAsset.String()
	sourceAssetsMap := map[string]xdr.Int64{}
	for i, sourceAsset := range sourceAssets {
		sourceAssetString := sourceAsset.String()
		sourceAssetsMap[sourceAssetString] = sourceAssetBalances[i]
	}

	searchState := &sellingGraphSearchState{
		graph:                  graph,
		destinationAsset:       destinationAsset,
		destinationAssetAmount: destinationAmount,
		ignoreOffersFrom:       sourceAccountID,
		targetAssets:           sourceAssetsMap,
		validateSourceBalance:  validateSourceBalance,
		paths:                  []Path{},
	}
	graph.lock.RLock()
	err := dfs(
		searchState,
		maxPathLength,
		map[string]bool{},
		[]xdr.Asset{},
		destinationAssetString,
		destinationAsset,
		destinationAmount,
	)
	lastLedger := graph.lastLedger
	graph.lock.RUnlock()
	if err != nil {
		return nil, lastLedger, errors.Wrap(err, "could not determine paths")
	}

	paths, err := sortAndFilterPaths(
		searchState.paths,
		maxAssetsPerPath,
		sortBySourceAsset,
	)
	return paths, lastLedger, err
}

// FindFixedPaths returns a list of payment paths where the source and destination
// assets are fixed. All returned payment paths will start by spending `amountToSpend`
// of `sourceAsset` and will end with some positive balance of `destinationAsset`.
// `sourceAccountID` is optional. if `sourceAccountID` is provided then no offers
// created by `sourceAccountID` will be considered when evaluating payment paths
func (graph *OrderBookGraph) FindFixedPaths(
	maxPathLength int,
	sourceAsset xdr.Asset,
	amountToSpend xdr.Int64,
	destinationAssets []xdr.Asset,
	maxAssetsPerPath int,
) ([]Path, uint32, error) {
	target := map[string]bool{}
	for _, destinationAsset := range destinationAssets {
		destinationAssetString := destinationAsset.String()
		target[destinationAssetString] = true
	}

	searchState := &buyingGraphSearchState{
		graph:             graph,
		sourceAsset:       sourceAsset,
		sourceAssetAmount: amountToSpend,
		targetAssets:      target,
		paths:             []Path{},
	}
	graph.lock.RLock()
	err := dfs(
		searchState,
		maxPathLength,
		map[string]bool{},
		[]xdr.Asset{},
		sourceAsset.String(),
		sourceAsset,
		amountToSpend,
	)
	lastLedger := graph.lastLedger
	graph.lock.RUnlock()
	if err != nil {
		return nil, lastLedger, errors.Wrap(err, "could not determine paths")
	}

	sort.Slice(searchState.paths, func(i, j int) bool {
		return searchState.paths[i].DestinationAmount > searchState.paths[j].DestinationAmount
	})

	paths, err := sortAndFilterPaths(
		searchState.paths,
		maxAssetsPerPath,
		sortByDestinationAsset,
	)
	return paths, lastLedger, err
}

// compareSourceAsset will group payment paths by `SourceAsset`
// paths which spend less `SourceAmount` will appear earlier in the sorting
// if there are multiple paths which spend the same `SourceAmount` then shorter payment paths
// will be prioritized
func compareSourceAsset(allPaths []Path, i, j int) bool {
	if allPaths[i].SourceAsset.Equals(allPaths[j].SourceAsset) {
		if allPaths[i].SourceAmount == allPaths[j].SourceAmount {
			return len(allPaths[i].InteriorNodes) < len(allPaths[j].InteriorNodes)
		}
		return allPaths[i].SourceAmount < allPaths[j].SourceAmount
	}
	return allPaths[i].SourceAssetString() < allPaths[j].SourceAssetString()
}

// compareDestinationAsset will group payment paths by `DestinationAsset`
// paths which deliver a higher `DestinationAmount` will appear earlier in the sorting
// if there are multiple paths which deliver the same `DestinationAmount` then shorter payment paths
// will be prioritized
func compareDestinationAsset(allPaths []Path, i, j int) bool {
	if allPaths[i].DestinationAsset.Equals(allPaths[j].DestinationAsset) {
		if allPaths[i].DestinationAmount == allPaths[j].DestinationAmount {
			return len(allPaths[i].InteriorNodes) < len(allPaths[j].InteriorNodes)
		}
		return allPaths[i].DestinationAmount > allPaths[j].DestinationAmount
	}
	return allPaths[i].DestinationAssetString() < allPaths[j].DestinationAssetString()
}

func sourceAssetEquals(p, otherPath Path) bool {
	return p.SourceAsset.Equals(otherPath.SourceAsset)
}

func destinationAssetEquals(p, otherPath Path) bool {
	return p.DestinationAsset.Equals(otherPath.DestinationAsset)
}

// sortAndFilterPaths sorts the given list of paths using `comparePaths`
// also, we limit the number of paths with the same asset to `maxPathsPerAsset`
func sortAndFilterPaths(
	allPaths []Path,
	maxPathsPerAsset int,
	sortType sortByType,
) ([]Path, error) {
	var comparePaths func([]Path, int, int) bool
	var assetsEqual func(Path, Path) bool

	switch sortType {
	case sortBySourceAsset:
		comparePaths = compareSourceAsset
		assetsEqual = sourceAssetEquals
	case sortByDestinationAsset:
		comparePaths = compareDestinationAsset
		assetsEqual = destinationAssetEquals
	default:
		return nil, errors.New("invalid sort by type")
	}

	sort.Slice(allPaths, func(i, j int) bool {
		return comparePaths(allPaths, i, j)
	})

	filtered := []Path{}
	countForAsset := 0
	for _, entry := range allPaths {
		if len(filtered) == 0 || !assetsEqual(filtered[len(filtered)-1], entry) {
			countForAsset = 1
			filtered = append(filtered, entry)
		} else if countForAsset < maxPathsPerAsset {
			countForAsset++
			filtered = append(filtered, entry)
		}
	}

	return filtered, nil
}
