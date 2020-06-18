package xdr

func (t Thresholds) MasterKeyWeight() byte {
	return t[0]
}

func (t Thresholds) ThresholdLow() byte {
	return t[1]
}

func (t Thresholds) ThresholdMedium() byte {
	return t[2]
}

func (t Thresholds) ThresholdHigh() byte {
	return t[3]
}
