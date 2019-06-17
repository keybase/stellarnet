package tickerdb

// InsertOrUpdateAsset inserts an Asset on the database (if new),
// or updates an existing one
func (s *TickerSession) InsertOrUpdateAsset(a *Asset, preserveFields []string) (err error) {
	return s.performUpsertQuery(*a, "assets", "assets_code_issuer_account", preserveFields)
}

// GetAssetByCodeAndIssuerAccount searches for an Asset with the given code
// and public key, and returns its ID in case it is found.
func (s *TickerSession) GetAssetByCodeAndIssuerAccount(
	code string,
	issuerAccount string,
) (found bool, id int32, err error) {
	var assets []Asset
	tbl := s.GetTable("assets")

	err = tbl.Select(
		&assets,
		"assets.code = ? AND assets.issuer_account = ?",
		code,
		issuerAccount,
	).Exec()
	if err != nil {
		return
	}

	if len(assets) > 0 {
		id = assets[0].ID
		found = true
	}
	return
}

// GetAllValidAssets returns a slice with all assets in the database
// with is_valid = true
func (s *TickerSession) GetAllValidAssets() (assets []Asset, err error) {
	tbl := s.GetTable("assets")

	err = tbl.Select(
		&assets,
		"assets.is_valid = TRUE",
	).Exec()

	return
}
