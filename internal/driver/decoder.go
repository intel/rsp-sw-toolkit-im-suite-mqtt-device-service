/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package driver

import (
	"encoding/hex"
	"fmt"
	"github.com/intel/rsp-sw-toolkit-im-suite-tagcode/bittag"
	"github.com/intel/rsp-sw-toolkit-im-suite-tagcode/epc"
	"github.com/pkg/errors"
	"strings"
)

type TagDecoder func(tagData []byte) (URI string, err error)

type NamedDecoder struct {
	Name string
	TagDecoder
}

type DecoderRing struct {
	Decoders []NamedDecoder
}

func (dr *DecoderRing) AddBitTagDecoder(authority, date string, widths []int) error {
	btd, err := bittag.NewDecoder(authority, date, widths)
	if err != nil {
		return err
	}

	decoder := func(tagData []byte) (string, error) {
		bitTag, err := btd.Decode(tagData)
		if err != nil {
			return "", err
		}
		return bitTag.URI(), nil
	}

	dr.Decoders = append(dr.Decoders, NamedDecoder{Name: btd.Prefix(), TagDecoder: decoder})
	return nil
}

func (dr *DecoderRing) AddSGTINDecoder(strict bool) {
	decoder := func(tagData []byte) (URI string, err error) {
		var s epc.SGTIN
		s, err = epc.DecodeSGTIN(tagData)
		if strict && err == nil {
			err = s.ValidateRanges()
		}
		if err == nil {
			URI = s.URI()
		}
		return
	}

	dr.Decoders = append(dr.Decoders, NamedDecoder{Name: "SGTIN", TagDecoder: decoder})
}

func (dr *DecoderRing) TagDataToURI(tagData string) (string, error) {
	tagDataBytes, err := hex.DecodeString(tagData)
	if err != nil {
		return "", errors.Wrap(err, "failed to decode tag hex data")
	}

	var decodingErrors []string
	for _, decoder := range dr.Decoders {
		tagUri, err := decoder.TagDecoder(tagDataBytes)
		if err == nil {
			return tagUri, nil
		}
		decodingErrors = append(decodingErrors, fmt.Sprintf("%s: %v",
			decoder.Name, err))
	}

	return "", errors.Errorf("no decoder successfully decoded the tag "+
		"data: individual decoder errors are as follows:\n%s",
		strings.Join(decodingErrors, "\n"))
}
