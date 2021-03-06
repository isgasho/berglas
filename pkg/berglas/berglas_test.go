// Copyright 2019 The Berglas Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package berglas

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"testing"

	uuid "github.com/satori/go.uuid"
)

func TestGsecretsIntegration(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping integration test (short)")
	}

	ctx := context.Background()

	bucket := os.Getenv("GOOGLE_CLOUD_BUCKET")
	if bucket == "" {
		t.Fatal("missing GOOGLE_CLOUD_BUCKET")
	}

	key := os.Getenv("GOOGLE_CLOUD_KMS_KEY")
	if key == "" {
		t.Fatal("missing GOOGLE_CLOUD_KMS_KEY")
	}

	sa := os.Getenv("GOOGLE_CLOUD_SERVICE_ACCOUNT")
	if sa == "" {
		t.Fatal("missing GOOGLE_CLOUD_SERVICE_ACCOUNT")
	}
	sa = fmt.Sprintf("serviceAccount:%s", sa)

	object := testUUID(t)

	c, err := New(ctx)
	if err != nil {
		t.Fatal(err)
	}

	original := []byte("original text")

	if err := c.Create(ctx, &CreateRequest{
		Bucket:    bucket,
		Object:    object,
		Key:       key,
		Plaintext: original,
	}); err != nil {
		t.Fatal(err)
	}

	secrets, err := c.List(ctx, &ListRequest{
		Bucket: bucket,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !testStringInclude(secrets, object) {
		t.Errorf("expected %q to include %q", secrets, object)
	}

	plaintext, err := c.Access(ctx, &AccessRequest{
		Bucket: bucket,
		Object: object,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(plaintext, original) {
		t.Errorf("expected %q to be %q", plaintext, original)
	}

	if err := c.Grant(ctx, &GrantRequest{
		Bucket:  bucket,
		Object:  object,
		Members: []string{sa},
	}); err != nil {
		t.Fatal(err)
	}

	if err := c.Revoke(ctx, &RevokeRequest{
		Bucket:  bucket,
		Object:  object,
		Members: []string{sa},
	}); err != nil {
		t.Fatal(err)
	}

	if err := c.Delete(ctx, &DeleteRequest{
		Bucket: bucket,
		Object: object,
	}); err != nil {
		t.Fatal(err)
	}
}

func testStringInclude(l []string, n string) bool {
	for _, v := range l {
		if n == v {
			return true
		}
	}
	return false
}

func testUUID(tb testing.TB) string {
	tb.Helper()

	u, err := uuid.NewV4()
	if err != nil {
		tb.Fatal(err)
	}
	return u.String()
}

func TestKMSKeyTrimVersion(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		i    string
		o    string
	}{
		{
			"malformed",
			"foo",
			"foo",
		},
		{
			"no_version",
			"projects/p/locations/l/keyRings/kr/cryptoKeys/ck",
			"projects/p/locations/l/keyRings/kr/cryptoKeys/ck",
		},
		{
			"version",
			"projects/p/locations/l/keyRings/kr/cryptoKeys/ck/cryptoKeyVersions/1",
			"projects/p/locations/l/keyRings/kr/cryptoKeys/ck",
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if act, exp := kmsKeyTrimVersion(tc.i), tc.o; act != exp {
				t.Errorf("expected %q to be %q", act, exp)
			}
		})
	}
}
