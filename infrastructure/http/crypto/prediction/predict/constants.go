package predict

const (
	chainIDBnbMainnet = 56
	protocolName      = "predict.fun CTF Exchange"
	protocolVersion   = "1"
	weiPrecision      = 1e18
	maxSalt           = 2147483648
	fiveMinutesSec    = 5 * 60
	defaultSlippageBps = 100

	ecdsaValidator = "0x845ADb2C711129d4f3966735eD98a9F09fC4cE57"
	zeroAddress    = "0x0000000000000000000000000000000000000000"
)

func exchangeContract(negRisk, yield bool) string {
	switch {
	case yield && negRisk:
		return "0x8A289d458f5a134bA40015085A8F50Ffb681B41d"
	case yield:
		return "0x6bEb5a40C032AFc305961162d8204CDA16DECFa5"
	case negRisk:
		return "0x365fb81bd4A24D6303cd2F19c349dE6894D8d58A"
	default:
		return "0x8BC070BEdAB741406F4B1Eb65A72bee27894B689"
	}
}