package actions

import (
	"context"
	"encoding/hex"
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/asaskevich/govalidator"
	"github.com/go-chi/chi"
	"github.com/gorilla/schema"

	"github.com/stellar/go/amount"
	"github.com/stellar/go/services/horizon/internal/assets"
	"github.com/stellar/go/services/horizon/internal/db2"
	"github.com/stellar/go/services/horizon/internal/httpx"
	"github.com/stellar/go/services/horizon/internal/ledger"
	hProblem "github.com/stellar/go/services/horizon/internal/render/problem"
	"github.com/stellar/go/services/horizon/internal/toid"
	"github.com/stellar/go/strkey"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/render/problem"
	"github.com/stellar/go/support/time"
	"github.com/stellar/go/xdr"
)

// TODO: move these constants to urlparam.go as we should parse the params with http handlers
// in the upper level package.
const (
	// ParamCursor is a query string param name
	ParamCursor = "cursor"
	// ParamOrder is a query string param name
	ParamOrder = "order"
	// ParamLimit is a query string param name
	ParamLimit = "limit"
	// LastLedgerHeaderName is the header which is set on all endpoints
	LastLedgerHeaderName = "Latest-Ledger"
)

type Opt int

const (
	// DisableCursorValidation disables cursor validation in GetPageQuery
	DisableCursorValidation Opt = iota
	// RequiredParam is used in Get* methods and defines a required parameter
	// (errors if value is empty).
	RequiredParam
	maxAssetCodeLength = 12
)

// HeaderWriter is an interface for setting HTTP response headers
type HeaderWriter interface {
	Header() http.Header
}

// SetLastLedgerHeader sets the Latest-Ledger header
func SetLastLedgerHeader(w HeaderWriter, lastLedger uint32) {
	w.Header().Set(LastLedgerHeaderName, strconv.FormatUint(uint64(lastLedger), 10))
}

// GetCursor retrieves a string from either the URLParams, form or query string.
// This method uses the priority (URLParams, Form, Query).
func (base *Base) GetCursor(name string) (cursor string) {
	if base.Err != nil {
		return ""
	}

	cursor, base.Err = GetCursor(base.R, name)

	return cursor
}

// GetCursor retrieves a string from either the URLParams, form or query string.
// This method uses the priority (URLParams, Form, Query).
func GetCursor(r *http.Request, name string) (string, error) {
	cursor, err := GetString(r, name)

	if err != nil {
		return "", err
	}

	if cursor == "now" {
		tid := toid.AfterLedger(ledger.CurrentState().HistoryLatest)
		cursor = tid.String()
	}

	if lastEventID := r.Header.Get("Last-Event-ID"); lastEventID != "" {
		cursor = lastEventID
	}

	// In case cursor is negative value, return InvalidField error
	cursorInt, err := strconv.Atoi(cursor)
	if err == nil && cursorInt < 0 {
		msg := fmt.Sprintf("the cursor %d is a negative number: ", cursorInt)

		return "", problem.MakeInvalidFieldProblem(
			name,
			errors.New(msg),
		)
	}

	return cursor, nil
}

func checkUTF8(name, value string) error {
	if !utf8.ValidString(value) {
		return problem.MakeInvalidFieldProblem(name, errors.New("invalid value"))
	}
	return nil
}

// GetStringFromURLParam retrieves a string from the URLParams.
func GetStringFromURLParam(r *http.Request, name string) (string, error) {
	fromURL, ok := GetURLParam(r, name)
	if ok {
		ret, err := url.PathUnescape(fromURL)
		if err != nil {
			return "", problem.MakeInvalidFieldProblem(name, err)
		}

		if err := checkUTF8(name, ret); err != nil {
			return "", err
		}
		return ret, nil
	}

	return "", nil
}

// GetStringFromURLParam retrieves a string from the URLParams.
func (base *Base) GetStringFromURLParam(name string) string {
	if base.Err != nil {
		return ""
	}

	var ret string
	ret, base.Err = GetString(base.R, name)
	return ret
}

// GetString retrieves a string from either the URLParams, form or query string.
// This method uses the priority (URLParams, Form, Query).
func GetString(r *http.Request, name string) (string, error) {
	fromURL, ok := GetURLParam(r, name)
	if ok {
		ret, err := url.PathUnescape(fromURL)
		if err != nil {
			return "", problem.MakeInvalidFieldProblem(name, err)
		}

		if err := checkUTF8(name, ret); err != nil {
			return "", err
		}

		return ret, nil
	}

	fromForm := r.FormValue(name)
	if fromForm != "" {
		if err := checkUTF8(name, fromForm); err != nil {
			return "", err
		}
		return fromForm, nil
	}

	value := r.URL.Query().Get(name)
	if err := checkUTF8(name, value); err != nil {
		return "", err
	}

	return value, nil
}

// GetString retrieves a string from either the URLParams, form or query string.
// This method uses the priority (URLParams, Form, Query).
func (base *Base) GetString(name string) (result string) {
	if base.Err != nil {
		return ""
	}

	result, base.Err = GetString(base.R, name)
	return result
}

// GetInt64 retrieves an int64 from the action parameter of the given name.
func GetInt64(r *http.Request, name string) (int64, error) {
	asStr, err := GetString(r, name)
	if err != nil {
		return 0, err
	}
	if asStr == "" {
		return 0, nil
	}

	asI64, err := strconv.ParseInt(asStr, 10, 64)
	if err != nil {
		return 0, problem.MakeInvalidFieldProblem(name, errors.New("unparseable value"))
	}

	return asI64, nil
}

// GetInt64 retrieves an int64 from the action parameter of the given name.
// Populates err if the value is not a valid int64
func (base *Base) GetInt64(name string) int64 {
	if base.Err != nil {
		return 0
	}

	var parsed int64
	parsed, base.Err = GetInt64(base.R, name)

	return parsed
}

// GetInt32 retrieves an int32 from the action parameter of the given name.
// Populates err if the value is not a valid int32
func (base *Base) GetInt32(name string) int32 {
	if base.Err != nil {
		return 0
	}

	asStr := base.GetString(name)
	if asStr == "" {
		return 0
	}

	asI64, err := strconv.ParseInt(asStr, 10, 32)
	if err != nil {
		base.SetInvalidField(name, errors.New("unparseable value"))
		return 0
	}

	return int32(asI64)
}

// GetBool retrieves a bool from the query parameter for the given name.
// Populates err if the value is not a valid bool.
// Defaults to `false` in case of an empty string. WARNING, do not change
// this behaviour without checking other modules, ex. this is critical
// that failed transactions are not included (`false`) by default.
func (base *Base) GetBool(name string) bool {
	if base.Err != nil {
		return false
	}

	asStr := base.R.URL.Query().Get(name)
	if asStr == "" {
		return false
	}

	if asStr == "true" {
		return true
	} else if asStr == "false" || asStr == "" {
		return false
	} else {
		base.SetInvalidField(name, errors.New("unparseable value"))
		return false
	}
}

// GetLimit retrieves a uint64 limit from the action parameter of the given
// name. Populates err if the value is not a valid limit.  Uses the provided
// default value if the limit parameter is a blank string.
func (base *Base) GetLimit(name string, def uint64, max uint64) (limit uint64) {
	if base.Err != nil {
		return 0
	}

	limit, base.Err = GetLimit(base.R, name, def, max)

	return limit
}

// GetLimit retrieves a uint64 limit from the action parameter of the given
// name. Populates err if the value is not a valid limit.  Uses the provided
// default value if the limit parameter is a blank string.
func GetLimit(r *http.Request, name string, def uint64, max uint64) (uint64, error) {
	limit, err := GetString(r, name)

	if err != nil {
		return 0, err
	}
	if limit == "" {
		return def, nil
	}

	asI64, err := strconv.ParseInt(limit, 10, 64)
	if err != nil {

		return 0, problem.MakeInvalidFieldProblem(name, errors.New("unparseable value"))
	}

	if asI64 <= 0 {
		err = errors.New("invalid limit: non-positive value provided")
	} else if asI64 > int64(max) {
		err = errors.Errorf("invalid limit: value provided that is over limit max of %d", max)
	}

	if err != nil {
		return 0, problem.MakeInvalidFieldProblem(name, err)
	}

	return uint64(asI64), nil
}

// GetPageQuery is a helper that returns a new db.PageQuery struct initialized
// using the results from a call to GetPagingParams()
func (base *Base) GetPageQuery(opts ...Opt) (result db2.PageQuery) {
	if base.Err != nil {
		return db2.PageQuery{}
	}

	result, base.Err = GetPageQuery(base.R, opts...)

	return result
}

// GetPageQuery is a helper that returns a new db.PageQuery struct initialized
// using the results from a call to GetPagingParams()
func GetPageQuery(r *http.Request, opts ...Opt) (db2.PageQuery, error) {
	disableCursorValidation := false
	for _, opt := range opts {
		if opt == DisableCursorValidation {
			disableCursorValidation = true
		}
	}

	cursor, err := GetCursor(r, ParamCursor)
	if err != nil {
		return db2.PageQuery{}, err
	}
	order, err := GetString(r, ParamOrder)
	if err != nil {
		return db2.PageQuery{}, err
	}
	limit, err := GetLimit(r, ParamLimit, db2.DefaultPageSize, db2.MaxPageSize)
	if err != nil {
		return db2.PageQuery{}, err
	}

	pageQuery, err := db2.NewPageQuery(cursor, !disableCursorValidation, order, limit)
	if err != nil {
		if invalidFieldError, ok := err.(*db2.InvalidFieldError); ok {
			err = problem.MakeInvalidFieldProblem(
				invalidFieldError.Name,
				err,
			)
		} else {
			err = problem.BadRequest
		}

		return db2.PageQuery{}, err
	}

	return pageQuery, nil
}

// GetAddress retrieves a stellar address.  It confirms the value loaded is a
// valid stellar address, setting an invalid field error if it is not.
func (base *Base) GetAddress(name string, opts ...Opt) (result string) {
	if base.Err != nil {
		return
	}

	requiredParam := false
	for _, opt := range opts {
		if opt == RequiredParam {
			requiredParam = true
		}
	}

	// We should check base.Err after this call. This is why it's better to remove base.Err.
	result = base.GetString(name)
	if result == "" && !requiredParam {
		return result
	}

	_, err := strkey.Decode(strkey.VersionByteAccountID, result)
	if err != nil {
		base.SetInvalidField(name, errors.New("invalid address"))
	}

	return result
}

// GetTransactionID retireves a transaction identifier by attempting to decode an hex-encoded,
// 64-digit lowercase string at the provided name.
func GetTransactionID(r *http.Request, name string) (string, error) {
	value, err := GetStringFromURLParam(r, name)
	if err != nil {
		return "", err
	}

	if value != "" {
		if _, err = hex.DecodeString(value); err != nil || len(value) != 64 || strings.ToLower(value) != value {
			return "", problem.MakeInvalidFieldProblem(name, errors.New("invalid hash format"))
		}
	}

	return value, nil
}

// GetAccountID retireves an xdr.AccountID by attempting to decode a stellar
// address at the provided name.
func GetAccountID(r *http.Request, name string) (xdr.AccountId, error) {
	value, err := GetString(r, name)
	if err != nil {
		return xdr.AccountId{}, err
	}

	result, err := xdr.AddressToAccountId(value)
	if err != nil {
		return result, problem.MakeInvalidFieldProblem(
			name,
			errors.New("invalid address"),
		)
	}

	return result, nil
}

// GetAccountID retireves an xdr.AccountID by attempting to decode a stellar
// address at the provided name.
func (base *Base) GetAccountID(name string) (result xdr.AccountId) {
	if base.Err != nil {
		return
	}

	result, base.Err = GetAccountID(base.R, name)
	return result
}

// GetPositiveAmount returns a native amount (i.e. 64-bit integer) by parsing
// the string at the provided name in accordance with the stellar client
// conventions. Renders error for negative amounts and zero.
func GetPositiveAmount(r *http.Request, fieldName string) (xdr.Int64, error) {
	amountString, err := GetString(r, fieldName)
	if err != nil {
		return 0, err
	}

	parsed, err := amount.Parse(amountString)
	if err != nil {
		return 0, problem.MakeInvalidFieldProblem(
			fieldName,
			errors.New("invalid amount"),
		)
	}

	if parsed <= 0 {
		return 0, problem.MakeInvalidFieldProblem(
			fieldName,
			errors.New("amount must be positive"),
		)
	}

	return parsed, nil
}

// getAssetType is a helper that returns a xdr.AssetType by reading a string
func getAssetType(r *http.Request, name string) (xdr.AssetType, error) {
	val, err := GetString(r, name)
	if err != nil {
		return xdr.AssetTypeAssetTypeNative, nil
	}

	t, err := assets.Parse(val)
	if err != nil {
		return t, problem.MakeInvalidFieldProblem(
			name,
			err,
		)
	}

	return t, nil
}

// GetAsset decodes an asset from the request fields prefixed by `prefix`.  To
// succeed, three prefixed fields must be present: asset_type, asset_code, and
// asset_issuer.
func GetAsset(r *http.Request, prefix string) (xdr.Asset, error) {
	var value interface{}
	t, err := getAssetType(r, prefix+"asset_type")
	if err != nil {
		return xdr.Asset{}, err
	}

	switch t {
	case xdr.AssetTypeAssetTypeCreditAlphanum4:
		a := xdr.AssetAlphaNum4{}
		a.Issuer, err = GetAccountID(r, prefix+"asset_issuer")
		if err != nil {
			return xdr.Asset{}, err
		}

		var code string
		code, err = GetString(r, prefix+"asset_code")
		if err != nil {
			return xdr.Asset{}, err
		}
		if len(code) > len(a.AssetCode) {
			err := problem.MakeInvalidFieldProblem(
				prefix+"asset_code",
				errors.New("code too long"),
			)
			return xdr.Asset{}, err
		}

		copy(a.AssetCode[:len(code)], []byte(code))
		value = a
	case xdr.AssetTypeAssetTypeCreditAlphanum12:
		a := xdr.AssetAlphaNum12{}
		a.Issuer, err = GetAccountID(r, prefix+"asset_issuer")
		if err != nil {
			return xdr.Asset{}, err
		}

		var code string
		code, err = GetString(r, prefix+"asset_code")
		if err != nil {
			return xdr.Asset{}, err
		}
		if len(code) > len(a.AssetCode) {
			err := problem.MakeInvalidFieldProblem(
				prefix+"asset_code",
				errors.New("code too long"),
			)
			return xdr.Asset{}, err
		}

		copy(a.AssetCode[:len(code)], []byte(code))
		value = a
	}

	result, err := xdr.NewAsset(t, value)
	if err != nil {
		panic(err)
	}

	return result, nil
}

// GetAsset decodes an asset from the request fields prefixed by `prefix`.  To
// succeed, three prefixed fields must be present: asset_type, asset_code, and
// asset_issuer.
func (base *Base) GetAsset(prefix string) (result xdr.Asset) {
	if base.Err != nil {
		return
	}

	result, base.Err = GetAsset(base.R, prefix)
	return result
}

// GetAssets parses a list of assets from a given request.
// The request parameter is expected to be a comma separated list of assets
// encoded in the format (Code:Issuer or "native") defined by SEP-0011
// https://github.com/stellar/stellar-protocol/pull/313
// If there is no request parameter present GetAssets will return an empty list of assets
func GetAssets(r *http.Request, name string) ([]xdr.Asset, error) {
	s, err := GetString(r, name)
	if err != nil {
		return nil, err
	}

	assets, err := xdr.BuildAssets(s)

	if err != nil {
		return nil, problem.MakeInvalidFieldProblem(
			name,
			err,
		)
	}

	return assets, nil
}

// MaybeGetAsset decodes an asset from the request fields as GetAsset does, but
// only if type field is populated. returns an additional boolean reflecting whether
// or not the decoding was performed
func MaybeGetAsset(r *http.Request, prefix string) (xdr.Asset, bool) {
	s, err := GetString(r, prefix+"asset_type")
	if err != nil || s == "" {
		return xdr.Asset{}, false
	}

	asset, err := GetAsset(r, prefix)
	if err != nil {
		return xdr.Asset{}, false
	}

	return asset, true
}

// MaybeGetAsset decodes an asset from the request fields as GetAsset does, but
// only if type field is populated. returns an additional boolean reflecting whether
// or not the decoding was performed
func (base *Base) MaybeGetAsset(prefix string) (xdr.Asset, bool) {
	if base.Err != nil {
		return xdr.Asset{}, false
	}

	return MaybeGetAsset(base.R, prefix)
}

// GetTimeMillis retrieves a TimeMillis from the action parameter of the given name.
// Populates err if the value is not a valid TimeMillis
func (base *Base) GetTimeMillis(name string) (timeMillis time.Millis) {
	if base.Err != nil {
		return
	}

	asStr := base.GetString(name)
	if asStr == "" {
		return
	}

	timeMillis, err := time.MillisFromString(asStr)
	if err != nil {
		base.SetInvalidField(name, err)
		return
	}

	return
}

// GetURLParam returns the corresponding URL parameter value from the request
// routing context and an additional boolean reflecting whether or not the
// param was found. This is ported from Chi since the Chi version returns ""
// for params not found. This is undesirable since "" also is a valid url param.
// Ref: https://github.com/go-chi/chi/blob/d132b31857e5922a2cc7963f4fcfd8f46b3f2e97/context.go#L69
func GetURLParam(r *http.Request, key string) (string, bool) {
	rctx := chi.RouteContext(r.Context())

	// Return immediately if keys does not match Values
	// This can happen when a named param is not specified.
	// This is a bug in chi: https://github.com/go-chi/chi/issues/426
	if len(rctx.URLParams.Keys) != len(rctx.URLParams.Values) {
		return "", false
	}

	for k := len(rctx.URLParams.Keys) - 1; k >= 0; k-- {
		if rctx.URLParams.Keys[k] == key {
			return rctx.URLParams.Values[k], true
		}
	}

	return "", false
}

// GetURLParam returns the corresponding URL parameter value from the request
// routing context and an additional boolean reflecting whether or not the
// param was found. This is ported from Chi since the Chi version returns ""
// for params not found. This is undesirable since "" also is a valid url param.
// Ref: https://github.com/go-chi/chi/blob/d132b31857e5922a2cc7963f4fcfd8f46b3f2e97/context.go#L69
func (base *Base) GetURLParam(key string) (string, bool) {
	return GetURLParam(base.R, key)
}

// SetInvalidField establishes an error response triggered by an invalid
// input field from the user.
func (base *Base) SetInvalidField(name string, reason error) {
	base.Err = problem.MakeInvalidFieldProblem(name, reason)
}

// Path returns the current action's path, as determined by the http.Request of
// this action
func (base *Base) Path() string {
	return base.R.URL.Path
}

// ValidateBodyType sets an error on the action if the requests Content-Type
//  is not `application/x-www-form-urlencoded`
func (base *Base) ValidateBodyType() {
	c := base.R.Header.Get("Content-Type")
	if c == "" {
		return
	}

	mt, _, err := mime.ParseMediaType(c)
	if err != nil {
		base.Err = err
		return
	}

	if mt != "application/x-www-form-urlencoded" && mt != "multipart/form-data" {
		base.Err = &hProblem.UnsupportedMediaType
	}
}

// FullURL returns a URL containing the information regarding the original
// request stored in the context.
func FullURL(ctx context.Context) *url.URL {
	url := httpx.BaseURL(ctx)
	r := httpx.RequestFromContext(ctx)
	if r != nil {
		url.Path = r.URL.Path
		url.RawQuery = r.URL.RawQuery
	}
	return url
}

// Note from chi: it is a good idea to set a Decoder instance as a package
// global, because it caches meta-data about structs, and an instance can be
// shared safely:
var decoder = schema.NewDecoder()

// GetParams fills a struct with values read from a request's query parameters.
func GetParams(dst interface{}, r *http.Request) error {
	query := r.URL.Query()

	// Merge chi's URLParams with URL Query Params. Given
	// `/accounts/{account_id}/transactions?foo=bar`, chi's URLParams will
	// contain `account_id` and URL Query params will contain `foo`.
	if rctx := chi.RouteContext(r.Context()); rctx != nil {
		for _, key := range rctx.URLParams.Keys {
			if key == "*" {
				continue
			}
			val := query.Get(key)
			if len(val) > 0 {
				return problem.MakeInvalidFieldProblem(
					key,
					errors.New("The parameter should not be included in the request"),
				)
			}

			param, _ := GetURLParam(r, key)
			query.Set(key, param)
		}
	}

	decoder.IgnoreUnknownKeys(true)
	if err := decoder.Decode(dst, query); err != nil {
		for k, e := range err.(schema.MultiError) {
			return problem.NewProblemWithInvalidField(
				problem.BadRequest,
				k,
				getSchemaErrorFieldMessage(k, e),
			)
		}
	}

	if _, err := govalidator.ValidateStruct(dst); err != nil {
		field, message := getErrorFieldMessage(err)
		err = problem.MakeInvalidFieldProblem(
			getSchemaTag(dst, field),
			errors.New(message),
		)

		return err
	}

	if v, ok := dst.(Validateable); ok {
		if err := v.Validate(); err != nil {
			return err
		}
	}

	return nil
}

func getSchemaTag(params interface{}, field string) string {
	v := reflect.ValueOf(params).Elem()
	qt := v.Type()
	f, _ := qt.FieldByName(field)
	return f.Tag.Get("schema")
}

// GetURIParams returns a list of query parameters for a given query struct
func GetURIParams(query interface{}, paginated bool) []string {
	params := getSchemaTags(reflect.ValueOf(query).Elem())
	if paginated {
		pagingParams := []string{
			"cursor",
			"limit",
			"order",
		}
		params = append(params, pagingParams...)
	}
	return params
}

func getSchemaTags(v reflect.Value) []string {
	qt := v.Type()
	fields := make([]string, 0, v.NumField())

	for i := 0; i < qt.NumField(); i++ {
		f := qt.Field(i)
		// Query structs can have embedded query structs
		if f.Type.Kind() == reflect.Struct {
			fields = append(fields, getSchemaTags(v.Field(i))...)
		} else {
			tag, ok := f.Tag.Lookup("schema")
			if ok {
				fields = append(fields, tag)
			}
		}
	}

	return fields
}

// ValidateAssetParams runs multiple checks on an asset query parameter
func ValidateAssetParams(aType, code, issuer, prefix string) error {
	// If asset type is not present but code or issuer are, then there is a
	// missing parameter and the request is unprocessable.
	if len(aType) == 0 {
		if len(code) > 0 || len(issuer) > 0 {
			return problem.MakeInvalidFieldProblem(
				prefix+"asset_type",
				errors.New("Missing parameter"),
			)
		}

		return nil
	}

	t, err := assets.Parse(aType)
	if err != nil {
		return problem.MakeInvalidFieldProblem(
			prefix+"asset_type",
			err,
		)
	}

	var validLen int
	switch t {
	case xdr.AssetTypeAssetTypeNative:
		// If asset type is native, issuer or code should not be included in the
		// request
		switch {
		case len(code) > 0:
			return problem.MakeInvalidFieldProblem(
				prefix+"asset_code",
				errors.New("native asset does not have a code"),
			)
		case len(issuer) > 0:
			return problem.MakeInvalidFieldProblem(
				prefix+"asset_issuer",
				errors.New("native asset does not have an issuer"),
			)
		}

		return nil
	case xdr.AssetTypeAssetTypeCreditAlphanum4:
		validLen = len(xdr.AssetAlphaNum4{}.AssetCode)
	case xdr.AssetTypeAssetTypeCreditAlphanum12:
		validLen = len(xdr.AssetAlphaNum12{}.AssetCode)
	}

	codeLen := len(code)
	if codeLen == 0 || codeLen > validLen {
		return problem.MakeInvalidFieldProblem(
			prefix+"asset_code",
			errors.New("Asset code must be 1-12 alphanumeric characters"),
		)
	}

	if len(issuer) == 0 {
		return problem.MakeInvalidFieldProblem(
			prefix+"asset_issuer",
			errors.New("Missing parameter"),
		)
	}

	return nil
}

// ValidateCursorWithinHistory compares the requested page of data against the
// ledger state of the history database.  In the event that the cursor is
// guaranteed to return no results, we return a 410 GONE http response.
func ValidateCursorWithinHistory(pq db2.PageQuery) error {
	// an ascending query should never return a gone response:  An ascending query
	// prior to known history should return results at the beginning of history,
	// and an ascending query beyond the end of history should not error out but
	// rather return an empty page (allowing code that tracks the procession of
	// some resource more easily).
	if pq.Order != "desc" {
		return nil
	}

	var cursor int64
	var err error

	// Checking for the presence of "-" to see whether we should use CursorInt64
	// or CursorInt64Pair
	if strings.Contains(pq.Cursor, "-") {
		cursor, _, err = pq.CursorInt64Pair("-")
	} else {
		cursor, err = pq.CursorInt64()
	}

	if err != nil {
		return problem.MakeInvalidFieldProblem("cursor", errors.New("invalid value"))
	}

	elder := toid.New(ledger.CurrentState().HistoryElder, 0, 0)

	if cursor <= elder.ToInt64() {
		return &hProblem.BeforeHistory
	}

	return nil
}

func countNonEmpty(params ...interface{}) (int, error) {
	count := 0

	for _, param := range params {
		switch param := param.(type) {
		default:
			return 0, errors.Errorf("unexpected type %T", param)
		case int32:
			if param != int32(0) {
				count++
			}
		case int64:
			if param != int64(0) {
				count++
			}
		case string:
			if param != "" {
				count++
			}
		}
	}

	return count, nil
}
