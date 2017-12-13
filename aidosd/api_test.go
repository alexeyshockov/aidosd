// Copyright (c) 2017 Aidos Developer

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package aidosd

import (
	"log"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"testing"

	"github.com/AidosKuneen/gadk"
)

func TestAPI(t *testing.T) {
	cdir, err := os.Getwd()
	if err != nil {
		t.Error(err)
	}
	fdb := filepath.Join(cdir, "aidosd.db")
	if err := os.Remove(fdb); err != nil {
		t.Log(err)
	}
	conf, err := Prepare("../aidosd.conf", []byte("test"))
	if err != nil {
		t.Error(err)
	}
	acc := make(map[string][]gadk.Address)
	vals := make(map[gadk.Address]int64)
	for _, ac := range []string{"ac1", "ac2", ""} {
		adr := newAddress(t, conf, ac)
		for _, a := range adr {
			acc[ac] = append(acc[ac], a)
			vals[a] = int64(rand.Int31())
		}
	}
	d1 := &dummy1{
		acc2adr: acc,
		vals:    vals,
		mtrytes: make(map[gadk.Trytes]gadk.Transaction),
		t:       t,
		isConf:  false,
	}
	conf.api = d1

	d1.setupTXs()
	testListAccounts(conf, d1)
	testlistaddressgroupings(conf, d1)
	testvalidateaddress1(conf, d1, "HZSMDORPCAFJJJNEEWZSP9OCQZAHCAVPBAXUTJKRCYZXMSNGERFZLQPNWOQQHK9RMJO9PNSVV9KR9DONH", true)
	testvalidateaddress1(conf, d1, "ZSMDORPCAFJJJNEEWZSP9OCQZAHCAVPBAXUTJKRCYZXMSNGERFZLQPNWOQQHK9RMJO9PNSVV9KR9DONH", false)
	testvalidateaddress2(conf, d1)
	if _, err := Walletnotify(conf); err != nil {
		t.Error(err)
	}
	for ac := range d1.acc2adr {
		testgetbalance(conf, d1, ac)
		testlisttransactions(conf, d1, ac)
	}
	d1.isConf = true
	for ac := range d1.acc2adr {
		testlisttransactions(conf, d1, ac)
	}
	testgetbalance2(conf, d1)
	testlisttransactions2(conf, d1)
	testgettransaction(conf, d1)
}

func testlistaddressgroupings(conf *Conf, d1 *dummy1) {
	req := &Request{
		JSONRPC: "1.0",
		ID:      "curltest",
		Method:  "listaddressgroupings",
		Params:  []interface{}{},
	}
	var resp Response
	if err := listaddressgroupings(conf, req, &resp); err != nil {
		d1.t.Error(err)
	}
	if resp.Error != nil {
		d1.t.Error(resp.Error)
	}
	result, ok := resp.Result.([][][]interface{})
	if !ok {
		d1.t.Error("result must be slice")
	}
	if len(result) != 1 {
		d1.t.Error("result length must be 1, but", len(result))
	}
	if len(result[0]) != 3*3 {
		d1.t.Error("result length must be 9,but", len(result[0]))
	}
	for i := range result[0] {
		adrstr, ok := result[0][i][0].(gadk.Trytes)
		if !ok {
			d1.t.Error("result[0][i][0] must be address")
		}
		adr, err := adrstr.ToAddress()
		if err != nil {
			d1.t.Error("invalid address")
		}
		acc, ok := result[0][i][2].(string)
		if !ok {
			d1.t.Error("result[0][i][2] must be string")
		}
		v, ok := d1.vals[adr]
		if !ok {
			d1.t.Error("invalid adrress")
		}
		val, ok := result[0][i][1].(float64)
		if !ok {
			d1.t.Error("result[0][i][1] must be float")
		}
		if float64(v)*0.00000001 != val {
			d1.t.Error("invalid value")
		}
		acc2, ok := d1.adr2acc[adr]
		if !ok {
			d1.t.Error("invalid address")
		}
		if acc2 != acc {
			d1.t.Error("invalid account")
		}
	}
}
func testvalidateaddress2(conf *Conf, d1 *dummy1) {
	var k gadk.Address
	for k = range d1.vals {
	}
	req := &Request{
		JSONRPC: "1.0",
		ID:      "curltest",
		Method:  "validateaddress",
		Params:  []interface{}{string(k.WithChecksum())},
	}
	var resp Response
	if err := validateaddress(conf, req, &resp); err != nil {
		d1.t.Error(err)
	}
	if resp.Error != nil {
		d1.t.Error(resp.Error)
	}
	result, ok := resp.Result.(*info)
	if !ok {
		d1.t.Error("result must be info struct")
	}
	if *result.Account != d1.adr2acc[k] || *result.IsCompressed ||
		*result.Pubkey != "" || *result.IsScript ||
		*result.IsWatchOnly {
		d1.t.Error("params must be empty")
	}
	if !result.IsValid {
		d1.t.Error("address must be valid")
	}
	if result.Address != string(k.WithChecksum()) {
		d1.t.Error("invalid address")
	}
	if result.ScriptPubKey != "" {
		d1.t.Error("scriptpubkey must be empty")
	}
	if !result.IsMine {
		d1.t.Error("address should be mine")
	}
}
func testvalidateaddress1(conf *Conf, d1 *dummy1, adr string, isValid bool) {
	req := &Request{
		JSONRPC: "1.0",
		ID:      "curltest",
		Method:  "validateaddress",
		Params:  []interface{}{adr},
	}
	var resp Response
	if err := validateaddress(conf, req, &resp); err != nil {
		d1.t.Error(err)
	}
	if resp.Error != nil {
		d1.t.Error(resp.Error)
	}
	result, ok := resp.Result.(*info)
	if !ok {
		d1.t.Error("result must be info struct")
	}
	if result.Account != nil || result.IsCompressed != nil ||
		result.Pubkey != nil || result.IsScript != nil ||
		result.IsWatchOnly != nil {
		d1.t.Error("params must be nil")
	}
	if result.IsValid != isValid {
		d1.t.Error("validity of address must be ", isValid)
	}
	if result.Address != adr {
		d1.t.Error("invalid address")
	}
	if result.ScriptPubKey != "" {
		d1.t.Error("scriptpubkey must be empty")
	}
	if result.IsMine {
		d1.t.Error("address should not be mine")
	}
}

func testgettransaction(conf *Conf, d1 *dummy1) {
	req := &Request{
		JSONRPC: "1.0",
		ID:      "curltest",
		Method:  "gettransaction",
		Params:  []interface{}{string(d1.bundle.Hash())},
	}
	var resp Response
	if err := gettransaction(conf, req, &resp); err != nil {
		d1.t.Error(err)
	}
	if resp.Error != nil {
		d1.t.Error(resp.Error)
	}
	tx, ok := resp.Result.(*tx)
	if !ok {
		d1.t.Error("result must be tx")
	}
	txs, amount := d1.list4Bundle()

	if tx.Amount != amount {
		d1.t.Error("amount is incorrect")
	}
	if tx.Fee != 0 ||
		len(tx.Walletconflicts) != 0 || tx.BIP125Replaceable != "no" || tx.Hex != "" {
		d1.t.Error("invalid dummy params")
	}
	if d1.isConf {
		if tx.Confirmations != 100000 {
			d1.t.Error("invalid confirmations")
		}
		if *tx.Blockhash != "" || *tx.Blockindex != 0 || *tx.Blocktime != tx.Time {
			d1.t.Error("invalid block params", *tx.Blockhash, *tx.Blockindex, *tx.Blocktime)
		}
	} else {
		if tx.Confirmations != 0 {
			d1.t.Error("invalid confirmations")
		}
		if tx.Blockhash != nil || tx.Blockindex != nil || tx.Blocktime != nil {
			d1.t.Error("invalid block params")
		}
	}
	if tx.Txid != d1.bundle.Hash() {
		d1.t.Error("invalid txid")
	}
	ok = false
	for _, txx := range txs {
		log.Println(txx.Timestamp.Unix())
		if tx.Time == txx.Timestamp.Unix() {
			ok = true
		}
	}
	if !ok {
		d1.t.Error("invalid time", tx.Time)
	}
	if tx.Time != tx.TimeReceived {
		d1.t.Error("invalid timereceived")
	}
	if len(txs) != len(tx.Details) {
		d1.t.Error("invalid number of length ")
	}
	for i, d := range tx.Details {
		if d.Address != txs[i].Address.WithChecksum() {
			d1.t.Error("invalid address", d.Address, txs[i].Address.WithChecksum())
		}
		adr, err := d.Address.ToAddress()
		if err != nil {
			d1.t.Error(err)
		}
		acc, ok := d1.adr2acc[adr]
		if !ok || acc != d.Account {
			d1.t.Error("invalid account")
		}
		if d.Amount > 0 && d.Category != "receive" {
			d1.t.Error("invalid category")
		}
		if d.Amount < 0 && d.Category != "send" {
			d1.t.Error("invalid category")
		}
		if d.Amount == 0 {
			d1.t.Error("invalid amount")
		}
		if d.Amount != float64(txs[i].Value)*0.00000001 {
			d1.t.Error("invalid amount")
		}
		if d.Fee != 0 {
			d1.t.Error("invalid dummy params")
		}
		if d.Category == "receive" && d.Abandoned != nil {
			d1.t.Error("invalid abandone")
		}
		if d.Category == "send" && (d.Abandoned == nil || *d.Abandoned != false) {
			d1.t.Error("invalid abandone")
		}
	}
}

func testgetbalance(conf *Conf, d1 *dummy1, ac string) {
	req := &Request{
		JSONRPC: "1.0",
		ID:      "curltest",
		Method:  "getbalance",
		Params:  []interface{}{ac},
	}

	var resp Response
	if err := getbalance(conf, req, &resp); err != nil {
		d1.t.Error(err)
	}
	if resp.Error != nil {
		d1.t.Error(resp.Error)
	}
	result, ok := resp.Result.(float64)
	if !ok {
		d1.t.Error("result must be float64")
	}
	var total int64
	for _, a := range d1.acc2adr[ac] {
		total += d1.vals[a]
	}
	if result != float64(total)*0.00000001 {
		d1.t.Error("invalid balance", result, ac, float64(total)*0.00000001, len(d1.acc2adr[""]))
	}
}

func testgetbalance2(conf *Conf, d1 *dummy1) {
	req := &Request{
		JSONRPC: "1.0",
		ID:      "curltest",
		Method:  "getbalance",
		Params:  []interface{}{},
	}

	var resp Response
	if err := getbalance(conf, req, &resp); err != nil {
		d1.t.Error(err)
	}
	if resp.Error != nil {
		d1.t.Error(resp.Error)
	}
	result, ok := resp.Result.(float64)
	if !ok {
		d1.t.Error("result must be float64")
	}
	var total int64
	for _, v := range d1.vals {
		total += v
	}
	if result != float64(total)*0.00000001 {
		d1.t.Error("invalid balance", result, float64(total)*0.00000001, len(d1.acc2adr[""]))
	}
}

func testlisttransactions(conf *Conf, d1 *dummy1, ac string) {
	skip := 1
	count := 2

	req := &Request{
		JSONRPC: "1.0",
		ID:      "curltest",
		Method:  "listtransactions",
		Params:  []interface{}{ac, float64(count), float64(skip)},
	}

	var resp Response
	if err := listtransactions(conf, req, &resp); err != nil {
		d1.t.Error(err)
	}
	if resp.Error != nil {
		d1.t.Error(resp.Error)
	}
	result, ok := resp.Result.([]*transaction)
	if !ok {
		d1.t.Error("result must be transaction struct")
	}
	if len(result) != count {
		d1.t.Error("invalid number of txs", len(result), ac)
	}
	txs := d1.list(ac, count, skip)
	var last int64 = math.MaxInt64
	for i, tx := range result {
		otx := txs[i]
		if *tx.Account != ac {
			d1.t.Error("invalid account")
		}
		if tx.Address != otx.Address.WithChecksum() {
			d1.t.Error("invalid address", tx.Address, otx.Address)
		}
		if tx.Amount > 0 && tx.Category != "receive" {
			d1.t.Error("invalid category")
		}
		if tx.Amount < 0 && tx.Category != "send" {
			d1.t.Error("invalid category")
		}
		if tx.Amount == 0 {
			d1.t.Error(" amount should not be 0")
		}
		if tx.Amount != float64(otx.Value)*0.00000001 {
			d1.t.Error("invalid amount", tx.Amount, ac)
		}
		if tx.Time != otx.Timestamp.Unix() {
			d1.t.Error("invalid time")
		}
		if tx.Time > last {
			d1.t.Error("invalid order")
		}
		last = tx.Time
		if tx.Txid != otx.Bundle {
			d1.t.Error("invalid txid")
		}
		conf := 100000
		if !d1.isConf {
			conf = 0
		}
		if tx.Confirmations != conf {
			d1.t.Error("invalid confirmations")
		}
		if tx.Vout != 0 || tx.Fee != 0 ||
			len(tx.Walletconflicts) != 0 || tx.BIP125Replaceable != "no" {
			d1.t.Error("invalid dummy params")
		}
		if d1.isConf {
			if *tx.Blockhash != "" || *tx.Blockindex != 0 || *tx.Blocktime != tx.Time {
				d1.t.Error("invalid block params")
			}
			if tx.Trusted != nil {
				d1.t.Error("invalid trusted")
			}
		} else {
			if tx.Blockhash != nil || tx.Blockindex != nil || tx.Blocktime != nil {
				d1.t.Error("invalid block params")
			}
			if *tx.Trusted != false {
				d1.t.Error("invalid trusted")
			}
		}
		if tx.Category == "receive" && tx.Abandoned != nil {
			d1.t.Error("invalid abandone")
		}
		if tx.Category == "send" && (tx.Abandoned == nil || *tx.Abandoned != false) {
			d1.t.Error("invalid abandone")
		}
	}
}

func testlisttransactions2(conf *Conf, d1 *dummy1) {
	req := &Request{
		JSONRPC: "1.0",
		ID:      "curltest",
		Method:  "listtransactions",
		Params:  []interface{}{},
	}

	var resp Response
	if err := listtransactions(conf, req, &resp); err != nil {
		d1.t.Error(err)
	}
	if resp.Error != nil {
		d1.t.Error(resp.Error)
	}
	result, ok := resp.Result.([]*transaction)
	if !ok {
		d1.t.Error("result must be transaction struct")
	}
	txs := d1.listall()
	if len(result) != 10 {
		d1.t.Error("invalid number of txs", len(result), len(txs))
	}
	var last int64 = math.MaxInt64
	for i, tx := range result {
		otx := txs[i]
		adr, err := tx.Address.ToAddress()
		if err != nil {
			d1.t.Error("invalid address")
		}
		if *tx.Account != d1.adr2acc[adr] {
			d1.t.Error("invalid account")
		}
		if tx.Address != otx.Address.WithChecksum() {
			d1.t.Error("invalid address", tx.Address, otx.Address)
		}
		if tx.Amount > 0 && tx.Category != "receive" {
			d1.t.Error("invalid category")
		}
		if tx.Amount < 0 && tx.Category != "send" {
			d1.t.Error("invalid category")
		}
		if tx.Amount == 0 {
			d1.t.Error(" amount should not be 0")
		}
		if tx.Amount != float64(otx.Value)*0.00000001 {
			d1.t.Error("invalid amount", tx.Amount)
		}
		if tx.Time != otx.Timestamp.Unix() {
			d1.t.Error("invalid time")
		}
		if tx.Time > last {
			d1.t.Error("invalid order")
		}
		last = tx.Time
		if tx.Txid != otx.Bundle {
			d1.t.Error("invalid txid")
		}
		conf := 100000
		if !d1.isConf {
			conf = 0
		}
		if tx.Confirmations != conf {
			d1.t.Error("invalid confirmations")
		}
		if tx.Vout != 0 || tx.Fee != 0 ||
			len(tx.Walletconflicts) != 0 || tx.BIP125Replaceable != "no" {
			d1.t.Error("invalid dummy params")
		}
		if d1.isConf {
			if *tx.Blockhash != "" || *tx.Blockindex != 0 || *tx.Blocktime != tx.Time {
				d1.t.Error("invalid block params")
			}
			if tx.Trusted != nil {
				d1.t.Error("invalid trusted")
			}
		} else {
			if tx.Blockhash != nil || tx.Blockindex != nil || tx.Blocktime != nil {
				d1.t.Error("invalid block params")
			}
			if *tx.Trusted != false {
				d1.t.Error("invalid trusted")
			}
		}
		if tx.Category == "receive" && tx.Abandoned != nil {
			d1.t.Error("invalid abandone")
		}
		if tx.Category == "send" && (tx.Abandoned == nil || *tx.Abandoned != false) {
			d1.t.Error("invalid abandone")
		}
	}
}

func testListAccounts(conf *Conf, d1 *dummy1) {
	req := &Request{
		JSONRPC: "1.0",
		ID:      "curltest",
		Method:  "listaccounts",
		Params:  []interface{}{},
	}
	var resp Response
	if err := listaccounts(conf, req, &resp); err != nil {
		d1.t.Error(err)
	}
	if resp.Error != nil {
		d1.t.Error(resp.Error)
	}
	result, ok := resp.Result.(map[string]float64)
	if !ok {
		d1.t.Error("result must be map")
	}
	if len(result) != len(d1.acc2adr) {
		d1.t.Error("result length is incorrect")
	}
	total := make(map[string]int64)
	for ac, as := range d1.acc2adr {
		for _, a := range as {
			total[ac] += d1.vals[a]
		}
	}
	for ac := range d1.acc2adr {
		if result[ac] != float64(total[ac])*0.00000001 {
			d1.t.Error("invalid balance", ac, result[ac], "must be", float64(total[ac])*0.00000001)
		}
	}
}

func newAddress(t *testing.T, conf *Conf, ac string) []gadk.Address {
	adrs := make([]gadk.Address, 3)
	req := &Request{
		JSONRPC: "1.0",
		ID:      "curltest",
		Method:  "getnewaddress",
		Params:  []interface{}{ac},
	}
	//test for default
	if ac == "" {
		req.Params = []interface{}{}
	}
	var resp Response
	for i := range adrs {
		if err := getnewaddress(conf, req, &resp); err != nil {
			t.Error(err)
		}
		if resp.Error != nil {
			t.Error("should not be error")
		}
		adrstr, ok := resp.Result.(gadk.Trytes)
		if !ok {
			t.Error("result must be trytes")
		}
		adr, err := adrstr.ToAddress()
		if err != nil {
			t.Error(err)
		}
		if err := adr.IsValid(); err != nil {
			t.Error(err)
		}
		adrs[i] = adr
	}
	return adrs
}
