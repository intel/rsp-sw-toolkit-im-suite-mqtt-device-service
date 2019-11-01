/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package main

import (
	expect "github.com/intel/rsp-sw-toolkit-im-suite-expect"
	gojsonschema "github.com/intel/rsp-sw-toolkit-im-suite-gojsonschema"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testSchemasDir(t *testing.T, dir string) {
	w := expect.WrapT(t)
	schemaData := w.ShouldHaveResult(ioutil.ReadFile(dir + "_meta_schema.json")).([]byte)
	metaSchema := w.Asf("create schema from %q", dir).ShouldHaveResult(
		gojsonschema.NewSchema(gojsonschema.NewBytesLoader(schemaData))).
	(*gojsonschema.Schema)

	w.As("walk " + dir).ShouldSucceed(filepath.Walk(dir,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				if path != dir {
					return filepath.SkipDir
				} else {
					return nil
				}
			}

			t.Run(info.Name(), func(tt *testing.T) {
				dw := expect.WrapT(tt).Asf("%q", path)
				if !strings.HasSuffix(info.Name(), "_schema.json") {
					dw.Errorf("file %q does not end in _schema.json", path)
					return
				}
				content := dw.StopOnMismatch().ShouldHaveResult(ioutil.ReadFile(path)).([]byte)
				r := dw.ShouldHaveResult(metaSchema.Validate(
					gojsonschema.NewBytesLoader(content))).(*gojsonschema.Result)
				dw.ShouldBeEmpty(r.Errors())
			})

			return nil
		}))
}

func TestSchemas(t *testing.T) {
	t.Run("test incoming schemas", func(t *testing.T) {
		testSchemasDir(t, "incoming")
	})
	t.Run("test responses schemas", func(t *testing.T) {
		testSchemasDir(t, "responses")
	})
}
