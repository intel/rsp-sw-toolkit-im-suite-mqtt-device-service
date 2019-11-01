/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package driver

import (
	expect "github.com/intel/rsp-sw-toolkit-im-suite-expect"
	"testing"
)

func TestDecoderRing_SGTIN(t *testing.T) {
	w := expect.WrapT(t)

	// these are two different EPC encodings of the _same_ SGTIN
	expectURI := "urn:epc:id:sgtin:0888446.067142.193853396487"
	epcSGTIN96 := "30143639F84191AD22901607"
	epcSGTIN198 := "36143639F8419198B966E1AB366E5B3470DC00000000000000"
	dr := DecoderRing{}
	dr.AddSGTINDecoder(true) // strict: requires valid SGTINs

	uriSGTIN96 := w.ShouldHaveResult(dr.TagDataToURI(epcSGTIN96)).(string)
	w.ShouldBeEqual(uriSGTIN96, expectURI)

	uriSGTIN198 := w.ShouldHaveResult(dr.TagDataToURI(epcSGTIN198)).(string)
	w.ShouldBeEqual(uriSGTIN198, "urn:epc:id:sgtin:0888446.067142.193853396487")

	// this is a valid SGTIN-198
	asciiSGTIN198 := "36143639F84191A465D9B37A176C5EB1769D72E557D52E5CBC"
	w.ShouldBeEqual(w.ShouldHaveResult(dr.TagDataToURI(asciiSGTIN198)).(string),
		"urn:epc:id:sgtin:0888446.067142.Hello!;1=1;'..*_*..%2F")
}

func TestDecoderRing_Custom(t *testing.T) {
	w := expect.WrapT(t)

	dr := DecoderRing{}
	w.ShouldSucceed(dr.AddBitTagDecoder(
		"test.com", "2019-01-01", []int{8, 48, 40}))

	custom := "0F00000000000C00000014D2"
	URI := w.ShouldHaveResult(dr.TagDataToURI(custom)).(string)
	w.As("URI").ShouldBeEqual(URI, "tag:test.com,2019-01-01:15.12.5330")

	w.As("not enough data").ShouldHaveError(dr.TagDataToURI(custom[:10]))
	URI = w.As("extra data is OK").ShouldHaveResult(
		dr.TagDataToURI(custom + "00")).(string)
	w.As("URI").ShouldBeEqual(URI, "tag:test.com,2019-01-01:15.12.5330")

	epcSGTIN96 := "30143639F84191AD22901607"
	URI = w.ShouldHaveResult(dr.TagDataToURI(epcSGTIN96)).(string)
	w.As("can't distinguish SGTINs").ShouldContainStr(
		URI, "tag:test.com,2019-01-01")
}

func TestDecoderRing_SGTINAndCustom(t *testing.T) {
	w := expect.WrapT(t)

	// create a ring with two decoders
	dr := DecoderRing{}
	dr.AddSGTINDecoder(true)
	w.ShouldSucceed(dr.AddBitTagDecoder(
		"test.com", "2019-01-01", []int{8, 48, 40}))

	sgtinURI := "urn:epc:id:sgtin:0888446.067142.193853396487"
	epcSGTIN96 := "30143639F84191AD22901607"
	epcSGTIN198 := "36143639F8419198B966E1AB366E5B3470DC00000000000000"

	uriSGTIN96 := w.ShouldHaveResult(dr.TagDataToURI(epcSGTIN96)).(string)
	w.ShouldBeEqual(uriSGTIN96, sgtinURI)

	uriSGTIN198 := w.ShouldHaveResult(dr.TagDataToURI(epcSGTIN198)).(string)
	w.ShouldBeEqual(uriSGTIN198, "urn:epc:id:sgtin:0888446.067142.193853396487")

	custom := "0F00000000000C00000014D2"
	URI := w.ShouldHaveResult(dr.TagDataToURI(custom)).(string)
	w.As("URI").ShouldBeEqual(URI, "tag:test.com,2019-01-01:15.12.5330")

	// not enough data for either
	w.As("can't be decoded").ShouldHaveError(dr.TagDataToURI(custom[:10]))

	// SGTINs shouldn't have null bytes in the middle
	almostSGTIN := "36143639F84191A465D9B37A176C5EB1769D72E557D5005CBC"
	URI = w.ShouldHaveResult(dr.TagDataToURI(almostSGTIN)).(string)
	w.ShouldContainStr(URI, "tag:test.com,2019-01-01")
}
