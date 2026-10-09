package tfupdate

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"unicode"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/minamijoyo/terraform-config-inspect/tfconfig"
	"github.com/minamijoyo/tfupdate/lock"
	"github.com/minamijoyo/tfupdate/tfregistry"
	"github.com/spf13/afero"
)

func TestNewLockUpdater(t *testing.T) {
	cases := []struct {
		platforms        []string
		tfregistryConfig tfregistry.Config
		want             *LockUpdater
		ok               bool
	}{
		{
			platforms:        []string{"darwin_arm64", "darwin_amd64", "linux_amd64"},
			tfregistryConfig: tfregistry.Config{},
			want: &LockUpdater{
				platforms:        []string{"darwin_arm64", "darwin_amd64", "linux_amd64"},
				tfregistryConfig: tfregistry.Config{},
			},
			ok: true,
		},
	}

	for _, tc := range cases {
		got, err := NewLockUpdater(tc.platforms, tc.tfregistryConfig)
		if tc.ok && err != nil {
			t.Errorf("NewLockUpdater() with platforms = %#v returns unexpected err: %+v", tc.platforms, err)
		}

		if !tc.ok && err == nil {
			t.Errorf("NewLockUpdater() with platforms = %#v expects to return an error, but no error", tc.platforms)
		}

		if tc.ok {
			gotUpdater, ok := got.(*LockUpdater)
			if !ok {
				t.Errorf("NewLockUpdater() returns %T, want *LockUpdater", got)
				continue
			}

			if diff := cmp.Diff(gotUpdater, tc.want,
				cmpopts.IgnoreFields(LockUpdater{}, "index"),
				cmp.AllowUnexported(LockUpdater{})); diff != "" {
				t.Errorf("NewLockUpdater() with platforms = %s mismatch (-got +want):\n%s", tc.platforms, diff)
			}
		}
	}
}

func TestLockUpdaterFullyQualifiedProviderAddress(t *testing.T) {
	cases := []struct {
		desc             string
		address          string
		tfregistryConfig tfregistry.Config
		want             string
		ok               bool
	}{
		{
			desc:             "normal provider address",
			address:          "hashicorp/null",
			tfregistryConfig: tfregistry.Config{},
			want:             "registry.terraform.io/hashicorp/null",
			ok:               true,
		},
		{
			desc:             "already fully qualified provider address",
			address:          "registry.terraform.io/hashicorp/null",
			tfregistryConfig: tfregistry.Config{},
			want:             "registry.terraform.io/hashicorp/null",
			ok:               true,
		},
		{
			desc:    "with custom BaseURL",
			address: "hashicorp/null",
			tfregistryConfig: tfregistry.Config{
				BaseURL: "https://registry.opentofu.org/",
			},
			want: "registry.opentofu.org/hashicorp/null",
			ok:   true,
		},
		{
			desc:    "with custom BaseURL and already fully qualified address",
			address: "registry.terraform.io/hashicorp/null",
			tfregistryConfig: tfregistry.Config{
				BaseURL: "https://registry.opentofu.org/",
			},
			want: "registry.opentofu.org/hashicorp/null",
			ok:   true,
		},
		{
			desc:             "unknown provider address (namespace omitted)",
			address:          "null",
			tfregistryConfig: tfregistry.Config{},
			want:             "",
			ok:               false,
		},
		{
			desc:             "legacy provider address (namespace as dash)",
			address:          "-/null",
			tfregistryConfig: tfregistry.Config{},
			want:             "",
			ok:               false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			u := &LockUpdater{
				tfregistryConfig: tc.tfregistryConfig,
			}

			got, err := u.fullyQualifiedProviderAddress(tc.address)
			if tc.ok && err != nil {
				t.Errorf("unexpected error: %s", err)
			}
			if !tc.ok && err == nil {
				t.Errorf("expected error, but got none")
			}
			if tc.ok && got != tc.want {
				t.Errorf("got: %s, want: %s", got, tc.want)
			}
		})
	}
}

func TestUpdateLockWithTerraformRegistry(t *testing.T) {
	platforms := []string{"darwin_arm64", "darwin_amd64", "linux_amd64"}
	pvs := []*lock.ProviderVersion{
		lock.NewMockProviderVersion(
			"hashicorp/aws",
			"5.4.0",
			[]string{"darwin_arm64", "darwin_amd64", "linux_amd64"},
			map[string]string{
				"terraform-provider-aws_5.4.0_darwin_arm64.zip": "h1:4eGsUS3r5eApQc19t8woc6d+sQLaOBaCSaK5GyGcWf0=",
				"terraform-provider-aws_5.4.0_linux_amd64.zip":  "h1:Jol4lNIzMrREQzUBSveCLX0iQLy7dm0OF+IYY2GKrhY=",
				"terraform-provider-aws_5.4.0_darwin_amd64.zip": "h1:ny1YPz2LiHTasDVNh6/HEvh1c9+TN/ftgAHh84bmy1E=",
			},
			map[string]string{
				"terraform-provider-aws_5.4.0_freebsd_arm.zip":   "zh:1db5f81089216831bb0fdff9ddc3772efa133397c66ec276bc75b96eec06e23f",
				"terraform-provider-aws_5.4.0_windows_386.zip":   "zh:26fe5fdf399192b5724d21854fbec650c158f8ee9eb1dc52a50f7da0f2bc07ac",
				"terraform-provider-aws_5.4.0_windows_amd64.zip": "zh:2946d9e333b1efe01588ee9f9771169fd3c3a4a7cb78ed8f91e8b3efd1a73850",
				"terraform-provider-aws_5.4.0_openbsd_arm.zip":   "zh:36ed69e8d3029332c8a52a70940f714fd579b9fd95f5569cc010ef11162f5bf7",
				"terraform-provider-aws_5.4.0_linux_386.zip":     "zh:46ba5ad1c3a3ef98c346356cfa4bdd9c2501c661c2513bb92f4413f2482fb24b",
				"terraform-provider-aws_5.4.0_linux_arm64.zip":   "zh:46c10aaa9672b54a14b0e0effdd6ecd9b8a539b3bfe273ac54111e7352a7bb4b",
				"terraform-provider-aws_5.4.0_darwin_arm64.zip":  "zh:47d7f57bcbe4fba2f960ab6c4228c5e9e586be2f233a8baa8962b51a63337179",
				"terraform-provider-aws_5.4.0_darwin_amd64.zip":  "zh:47e41c198439ba1c4d933f808b6f47e518f8f0aae25ca42abcac97f149121e90",
				"terraform-provider-aws_5.4.0_openbsd_386.zip":   "zh:526c5834de71654ee14039cb973322bf5032cb684a2a113b48fb48a0584f46f3",
				"terraform-provider-aws_5.4.0_linux_arm.zip":     "zh:6169316517b95677819ba2904dcea204fb9b55e868348e906af9164104fe7198",
				"terraform-provider-aws_5.4.0_freebsd_amd64.zip": "zh:7c063ef2b8d69a8db7e8bf0dcd45793ede22b259b30464ed114d330df304cdbb",
				"terraform-provider-aws_5.4.0_freebsd_386.zip":   "zh:87c4f2faca636715a08be3121d26b3354415401eab89349077ca9436a0822c23",
				"terraform-provider-aws_5.4.0_manifest.json":     "zh:9b12af85486a96aedd8d7984b0ff811a4b42e3d88dad1a3fb4c0b580d04fa425",
				"terraform-provider-aws_5.4.0_openbsd_amd64.zip": "zh:b184b8a268f45258edd27d389ca793708f1bc3ee4d6706d154a45e93deaddde1",
				"terraform-provider-aws_5.4.0_linux_amd64.zip":   "zh:ba1a998cbf4b639fa3e04b9069f0f5a289662457940726a8a51c81df400aa852",
			},
		),
		lock.NewMockProviderVersion(
			"hashicorp/null",
			"3.2.1",
			[]string{"darwin_arm64", "darwin_amd64", "linux_amd64"},
			map[string]string{
				"terraform-provider-null_3.2.1_linux_amd64.zip":  "h1:FbGfc+muBsC17Ohy5g806iuI1hQc4SIexpYCrQHQd8w=",
				"terraform-provider-null_3.2.1_darwin_amd64.zip": "h1:tSj1mL6OQ8ILGqR2mDu7OYYYWf+hoir0pf9KAQ8IzO8=",
				"terraform-provider-null_3.2.1_darwin_arm64.zip": "h1:ydA0/SNRVB1o95btfshvYsmxA+jZFRZcvKzZSB+4S1M=",
			},
			map[string]string{
				"terraform-provider-null_3.2.1_freebsd_arm.zip":   "zh:58ed64389620cc7b82f01332e27723856422820cfd302e304b5f6c3436fb9840",
				"terraform-provider-null_3.2.1_windows_386.zip":   "zh:62a5cc82c3b2ddef7ef3a6f2fedb7b9b3deff4ab7b414938b08e51d6e8be87cb",
				"terraform-provider-null_3.2.1_darwin_amd64.zip":  "zh:63cff4de03af983175a7e37e52d4bd89d990be256b16b5c7f919aff5ad485aa5",
				"terraform-provider-null_3.2.1_linux_amd64.zip":   "zh:74cb22c6700e48486b7cabefa10b33b801dfcab56f1a6ac9b6624531f3d36ea3",
				"terraform-provider-null_3.2.1_manifest.json":     "zh:78d5eefdd9e494defcb3c68d282b8f96630502cac21d1ea161f53cfe9bb483b3",
				"terraform-provider-null_3.2.1_windows_amd64.zip": "zh:79e553aff77f1cfa9012a2218b8238dd672ea5e1b2924775ac9ac24d2a75c238",
				"terraform-provider-null_3.2.1_freebsd_amd64.zip": "zh:a1e06ddda0b5ac48f7e7c7d59e1ab5a4073bbcf876c73c0299e4610ed53859dc",
				"terraform-provider-null_3.2.1_linux_arm.zip":     "zh:c37a97090f1a82222925d45d84483b2aa702ef7ab66532af6cbcfb567818b970",
				"terraform-provider-null_3.2.1_darwin_arm64.zip":  "zh:e4453fbebf90c53ca3323a92e7ca0f9961427d2f0ce0d2b65523cc04d5d999c2",
				"terraform-provider-null_3.2.1_linux_386.zip":     "zh:e80a746921946d8b6761e77305b752ad188da60688cfd2059322875d363be5f5",
				"terraform-provider-null_3.2.1_freebsd_386.zip":   "zh:fbdb892d9822ed0e4cb60f2fedbdbb556e4da0d88d3b942ae963ed6ff091e48f",
				"terraform-provider-null_3.2.1_linux_arm64.zip":   "zh:fca01a623d90d0cad0843102f9b8b9fe0d3ff8244593bd817f126582b52dd694",
			},
		),
	}

	cases := []struct {
		desc     string
		src      string
		lockfile string
		want     string
		ok       bool
	}{
		{
			desc: "simple",
			src: `
terraform {
  required_version = "1.5.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "5.4.0"
    }

    null = {
      source  = "hashicorp/null"
      version = "3.2.1"
    }

    github = {
      source  = "integrations/github"
      version = "4.28.0"
    }
  }
}
`,
			lockfile: `
# This file is maintained automatically by "terraform init".
# Manual edits may be lost in future updates.

provider "registry.terraform.io/hashicorp/aws" {
  version     = "5.4.0"
  constraints = "5.4.0"
  hashes = [
    "h1:4eGsUS3r5eApQc19t8woc6d+sQLaOBaCSaK5GyGcWf0=",
    "h1:Jol4lNIzMrREQzUBSveCLX0iQLy7dm0OF+IYY2GKrhY=",
    "h1:ny1YPz2LiHTasDVNh6/HEvh1c9+TN/ftgAHh84bmy1E=",
    "zh:1db5f81089216831bb0fdff9ddc3772efa133397c66ec276bc75b96eec06e23f",
    "zh:26fe5fdf399192b5724d21854fbec650c158f8ee9eb1dc52a50f7da0f2bc07ac",
    "zh:2946d9e333b1efe01588ee9f9771169fd3c3a4a7cb78ed8f91e8b3efd1a73850",
    "zh:36ed69e8d3029332c8a52a70940f714fd579b9fd95f5569cc010ef11162f5bf7",
    "zh:46ba5ad1c3a3ef98c346356cfa4bdd9c2501c661c2513bb92f4413f2482fb24b",
    "zh:46c10aaa9672b54a14b0e0effdd6ecd9b8a539b3bfe273ac54111e7352a7bb4b",
    "zh:47d7f57bcbe4fba2f960ab6c4228c5e9e586be2f233a8baa8962b51a63337179",
    "zh:47e41c198439ba1c4d933f808b6f47e518f8f0aae25ca42abcac97f149121e90",
    "zh:526c5834de71654ee14039cb973322bf5032cb684a2a113b48fb48a0584f46f3",
    "zh:6169316517b95677819ba2904dcea204fb9b55e868348e906af9164104fe7198",
    "zh:7c063ef2b8d69a8db7e8bf0dcd45793ede22b259b30464ed114d330df304cdbb",
    "zh:87c4f2faca636715a08be3121d26b3354415401eab89349077ca9436a0822c23",
    "zh:9b12af85486a96aedd8d7984b0ff811a4b42e3d88dad1a3fb4c0b580d04fa425",
    "zh:b184b8a268f45258edd27d389ca793708f1bc3ee4d6706d154a45e93deaddde1",
    "zh:ba1a998cbf4b639fa3e04b9069f0f5a289662457940726a8a51c81df400aa852",
  ]
}

provider "registry.terraform.io/hashicorp/null" {
  version     = "3.1.1"
  constraints = "3.1.1"
  hashes = [
    "h1:71sNUDvmiJcijsvfXpiLCz0lXIBSsEJjMxljt7hxMhw=",
    "h1:Pctug/s/2Hg5FJqjYcTM0kPyx3AoYK1MpRWO0T9V2ns=",
    "h1:YvH6gTaQzGdNv+SKTZujU1O0bO+Pw6vJHOPhqgN8XNs=",
    "zh:063466f41f1d9fd0dd93722840c1314f046d8760b1812fa67c34de0afcba5597",
    "zh:08c058e367de6debdad35fc24d97131c7cf75103baec8279aba3506a08b53faf",
    "zh:73ce6dff935150d6ddc6ac4a10071e02647d10175c173cfe5dca81f3d13d8afe",
    "zh:78d5eefdd9e494defcb3c68d282b8f96630502cac21d1ea161f53cfe9bb483b3",
    "zh:8fdd792a626413502e68c195f2097352bdc6a0df694f7df350ed784741eb587e",
    "zh:976bbaf268cb497400fd5b3c774d218f3933271864345f18deebe4dcbfcd6afa",
    "zh:b21b78ca581f98f4cdb7a366b03ae9db23a73dfa7df12c533d7c19b68e9e72e5",
    "zh:b7fc0c1615dbdb1d6fd4abb9c7dc7da286631f7ca2299fb9cd4664258ccfbff4",
    "zh:d1efc942b2c44345e0c29bc976594cb7278c38cfb8897b344669eafbc3cddf46",
    "zh:e356c245b3cd9d4789bab010893566acace682d7db877e52d40fc4ca34a50924",
    "zh:ea98802ba92fcfa8cf12cbce2e9e7ebe999afbf8ed47fa45fc847a098d89468b",
    "zh:eff8872458806499889f6927b5d954560f3d74bf20b6043409edf94d26cd906f",
  ]
}

provider "registry.terraform.io/integrations/github" {
  version     = "4.28.0"
  constraints = "4.28.0"
  hashes = [
    "h1:GMp4fa/6ZxeM8c8O20rZ2jpXXkBqK09oMBtWQ1WwPCo=",
    "h1:PRj9EXEvLgKTmQHKUtzIG28goXJX74aRt0b/4JH6qN8=",
    "h1:vAZrilSL9rq6bXb97dl06QRohtcc0btFQzFF5dHinI0=",
    "zh:125a1decda8a9d4c6d18010f3c66943c868da9e984298c0e2f9dfd240ec660ec",
    "zh:23a4cb334a2fbead38264f434c81e52cb52fb115cbc39537fefc9c22aaecdf35",
    "zh:3cf793b1d0bc30a703315c6ecb6bb2f36d14ed310dec7e300ae4a4a3a470aafe",
    "zh:47cb06845730df19256882272690221db8314199a34012ac7e690e0550ca9404",
    "zh:5d6e76624d60b6298ee47c10cc262adc9f361f4648f40faf81ee3a8d6beaad31",
    "zh:6415a5c6ba5b28f1f410845706cff0390718113f7d987aaa011553b041ba2005",
    "zh:70ce96d7aa424aef47d4b049d39aff036ae6377dacd5c077501eb0f353901cc6",
    "zh:9803fc59cf71ea629308773d429c9ca00985acdcc02d9755fc59900bcf6d1d00",
    "zh:a9a505f208f569ee44a0a6a7c975e3441bb8d61dbf9831c44c3be299e2cf1a21",
    "zh:a9d9a17b0618ea14f9fa49dfc1329b01473a9d708011fca32cd01b474051d169",
    "zh:bce0257085a5d6c9f0e6cdd5a704c50286c5382f840384a2a50c69d8488652bf",
    "zh:d7272bb396e67ff22d7f4628d152fa66610cf7507a4e63d72ef50fde651e39bf",
    "zh:e2aab496c17acb8c2bdd5af9e830e9f91f869d9fc173e6dd65b7475e8baa6f82",
    "zh:ea20984a5386fc4a6856eed58d261c5124fc8ca72bc6ee142c1092036a3c8360",
  ]
}
`,
			want: `
# This file is maintained automatically by "terraform init".
# Manual edits may be lost in future updates.

provider "registry.terraform.io/hashicorp/aws" {
  version     = "5.4.0"
  constraints = "5.4.0"
  hashes = [
    "h1:4eGsUS3r5eApQc19t8woc6d+sQLaOBaCSaK5GyGcWf0=",
    "h1:Jol4lNIzMrREQzUBSveCLX0iQLy7dm0OF+IYY2GKrhY=",
    "h1:ny1YPz2LiHTasDVNh6/HEvh1c9+TN/ftgAHh84bmy1E=",
    "zh:1db5f81089216831bb0fdff9ddc3772efa133397c66ec276bc75b96eec06e23f",
    "zh:26fe5fdf399192b5724d21854fbec650c158f8ee9eb1dc52a50f7da0f2bc07ac",
    "zh:2946d9e333b1efe01588ee9f9771169fd3c3a4a7cb78ed8f91e8b3efd1a73850",
    "zh:36ed69e8d3029332c8a52a70940f714fd579b9fd95f5569cc010ef11162f5bf7",
    "zh:46ba5ad1c3a3ef98c346356cfa4bdd9c2501c661c2513bb92f4413f2482fb24b",
    "zh:46c10aaa9672b54a14b0e0effdd6ecd9b8a539b3bfe273ac54111e7352a7bb4b",
    "zh:47d7f57bcbe4fba2f960ab6c4228c5e9e586be2f233a8baa8962b51a63337179",
    "zh:47e41c198439ba1c4d933f808b6f47e518f8f0aae25ca42abcac97f149121e90",
    "zh:526c5834de71654ee14039cb973322bf5032cb684a2a113b48fb48a0584f46f3",
    "zh:6169316517b95677819ba2904dcea204fb9b55e868348e906af9164104fe7198",
    "zh:7c063ef2b8d69a8db7e8bf0dcd45793ede22b259b30464ed114d330df304cdbb",
    "zh:87c4f2faca636715a08be3121d26b3354415401eab89349077ca9436a0822c23",
    "zh:9b12af85486a96aedd8d7984b0ff811a4b42e3d88dad1a3fb4c0b580d04fa425",
    "zh:b184b8a268f45258edd27d389ca793708f1bc3ee4d6706d154a45e93deaddde1",
    "zh:ba1a998cbf4b639fa3e04b9069f0f5a289662457940726a8a51c81df400aa852",
  ]
}

provider "registry.terraform.io/hashicorp/null" {
  version     = "3.2.1"
  constraints = "3.2.1"
  hashes = [
    "h1:FbGfc+muBsC17Ohy5g806iuI1hQc4SIexpYCrQHQd8w=",
    "h1:tSj1mL6OQ8ILGqR2mDu7OYYYWf+hoir0pf9KAQ8IzO8=",
    "h1:ydA0/SNRVB1o95btfshvYsmxA+jZFRZcvKzZSB+4S1M=",
    "zh:58ed64389620cc7b82f01332e27723856422820cfd302e304b5f6c3436fb9840",
    "zh:62a5cc82c3b2ddef7ef3a6f2fedb7b9b3deff4ab7b414938b08e51d6e8be87cb",
    "zh:63cff4de03af983175a7e37e52d4bd89d990be256b16b5c7f919aff5ad485aa5",
    "zh:74cb22c6700e48486b7cabefa10b33b801dfcab56f1a6ac9b6624531f3d36ea3",
    "zh:78d5eefdd9e494defcb3c68d282b8f96630502cac21d1ea161f53cfe9bb483b3",
    "zh:79e553aff77f1cfa9012a2218b8238dd672ea5e1b2924775ac9ac24d2a75c238",
    "zh:a1e06ddda0b5ac48f7e7c7d59e1ab5a4073bbcf876c73c0299e4610ed53859dc",
    "zh:c37a97090f1a82222925d45d84483b2aa702ef7ab66532af6cbcfb567818b970",
    "zh:e4453fbebf90c53ca3323a92e7ca0f9961427d2f0ce0d2b65523cc04d5d999c2",
    "zh:e80a746921946d8b6761e77305b752ad188da60688cfd2059322875d363be5f5",
    "zh:fbdb892d9822ed0e4cb60f2fedbdbb556e4da0d88d3b942ae963ed6ff091e48f",
    "zh:fca01a623d90d0cad0843102f9b8b9fe0d3ff8244593bd817f126582b52dd694",
  ]
}

provider "registry.terraform.io/integrations/github" {
  version     = "4.28.0"
  constraints = "4.28.0"
  hashes = [
    "h1:GMp4fa/6ZxeM8c8O20rZ2jpXXkBqK09oMBtWQ1WwPCo=",
    "h1:PRj9EXEvLgKTmQHKUtzIG28goXJX74aRt0b/4JH6qN8=",
    "h1:vAZrilSL9rq6bXb97dl06QRohtcc0btFQzFF5dHinI0=",
    "zh:125a1decda8a9d4c6d18010f3c66943c868da9e984298c0e2f9dfd240ec660ec",
    "zh:23a4cb334a2fbead38264f434c81e52cb52fb115cbc39537fefc9c22aaecdf35",
    "zh:3cf793b1d0bc30a703315c6ecb6bb2f36d14ed310dec7e300ae4a4a3a470aafe",
    "zh:47cb06845730df19256882272690221db8314199a34012ac7e690e0550ca9404",
    "zh:5d6e76624d60b6298ee47c10cc262adc9f361f4648f40faf81ee3a8d6beaad31",
    "zh:6415a5c6ba5b28f1f410845706cff0390718113f7d987aaa011553b041ba2005",
    "zh:70ce96d7aa424aef47d4b049d39aff036ae6377dacd5c077501eb0f353901cc6",
    "zh:9803fc59cf71ea629308773d429c9ca00985acdcc02d9755fc59900bcf6d1d00",
    "zh:a9a505f208f569ee44a0a6a7c975e3441bb8d61dbf9831c44c3be299e2cf1a21",
    "zh:a9d9a17b0618ea14f9fa49dfc1329b01473a9d708011fca32cd01b474051d169",
    "zh:bce0257085a5d6c9f0e6cdd5a704c50286c5382f840384a2a50c69d8488652bf",
    "zh:d7272bb396e67ff22d7f4628d152fa66610cf7507a4e63d72ef50fde651e39bf",
    "zh:e2aab496c17acb8c2bdd5af9e830e9f91f869d9fc173e6dd65b7475e8baa6f82",
    "zh:ea20984a5386fc4a6856eed58d261c5124fc8ca72bc6ee142c1092036a3c8360",
  ]
}
`,
			ok: true,
		},
		{
			desc: "noop",
			src: `
terraform {
  required_version = "1.5.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "5.4.0"
    }

    null = {
      source  = "hashicorp/null"
      version = "3.2.1"
    }

    github = {
      source  = "integrations/github"
      version = "4.28.0"
    }
  }
}
`,
			lockfile: `
# This file is maintained automatically by "terraform init".
# Manual edits may be lost in future updates.

provider "registry.terraform.io/hashicorp/aws" {
  version     = "5.4.0"
  constraints = "5.4.0"
  hashes = [
    "h1:4eGsUS3r5eApQc19t8woc6d+sQLaOBaCSaK5GyGcWf0=",
    "h1:Jol4lNIzMrREQzUBSveCLX0iQLy7dm0OF+IYY2GKrhY=",
    "h1:ny1YPz2LiHTasDVNh6/HEvh1c9+TN/ftgAHh84bmy1E=",
    "zh:1db5f81089216831bb0fdff9ddc3772efa133397c66ec276bc75b96eec06e23f",
    "zh:26fe5fdf399192b5724d21854fbec650c158f8ee9eb1dc52a50f7da0f2bc07ac",
    "zh:2946d9e333b1efe01588ee9f9771169fd3c3a4a7cb78ed8f91e8b3efd1a73850",
    "zh:36ed69e8d3029332c8a52a70940f714fd579b9fd95f5569cc010ef11162f5bf7",
    "zh:46ba5ad1c3a3ef98c346356cfa4bdd9c2501c661c2513bb92f4413f2482fb24b",
    "zh:46c10aaa9672b54a14b0e0effdd6ecd9b8a539b3bfe273ac54111e7352a7bb4b",
    "zh:47d7f57bcbe4fba2f960ab6c4228c5e9e586be2f233a8baa8962b51a63337179",
    "zh:47e41c198439ba1c4d933f808b6f47e518f8f0aae25ca42abcac97f149121e90",
    "zh:526c5834de71654ee14039cb973322bf5032cb684a2a113b48fb48a0584f46f3",
    "zh:6169316517b95677819ba2904dcea204fb9b55e868348e906af9164104fe7198",
    "zh:7c063ef2b8d69a8db7e8bf0dcd45793ede22b259b30464ed114d330df304cdbb",
    "zh:87c4f2faca636715a08be3121d26b3354415401eab89349077ca9436a0822c23",
    "zh:9b12af85486a96aedd8d7984b0ff811a4b42e3d88dad1a3fb4c0b580d04fa425",
    "zh:b184b8a268f45258edd27d389ca793708f1bc3ee4d6706d154a45e93deaddde1",
    "zh:ba1a998cbf4b639fa3e04b9069f0f5a289662457940726a8a51c81df400aa852",
  ]
}

provider "registry.terraform.io/hashicorp/null" {
  version     = "3.2.1"
  constraints = "3.2.1"
  hashes = [
    "h1:FbGfc+muBsC17Ohy5g806iuI1hQc4SIexpYCrQHQd8w=",
    "h1:tSj1mL6OQ8ILGqR2mDu7OYYYWf+hoir0pf9KAQ8IzO8=",
    "h1:ydA0/SNRVB1o95btfshvYsmxA+jZFRZcvKzZSB+4S1M=",
    "zh:58ed64389620cc7b82f01332e27723856422820cfd302e304b5f6c3436fb9840",
    "zh:62a5cc82c3b2ddef7ef3a6f2fedb7b9b3deff4ab7b414938b08e51d6e8be87cb",
    "zh:63cff4de03af983175a7e37e52d4bd89d990be256b16b5c7f919aff5ad485aa5",
    "zh:74cb22c6700e48486b7cabefa10b33b801dfcab56f1a6ac9b6624531f3d36ea3",
    "zh:78d5eefdd9e494defcb3c68d282b8f96630502cac21d1ea161f53cfe9bb483b3",
    "zh:79e553aff77f1cfa9012a2218b8238dd672ea5e1b2924775ac9ac24d2a75c238",
    "zh:a1e06ddda0b5ac48f7e7c7d59e1ab5a4073bbcf876c73c0299e4610ed53859dc",
    "zh:c37a97090f1a82222925d45d84483b2aa702ef7ab66532af6cbcfb567818b970",
    "zh:e4453fbebf90c53ca3323a92e7ca0f9961427d2f0ce0d2b65523cc04d5d999c2",
    "zh:e80a746921946d8b6761e77305b752ad188da60688cfd2059322875d363be5f5",
    "zh:fbdb892d9822ed0e4cb60f2fedbdbb556e4da0d88d3b942ae963ed6ff091e48f",
    "zh:fca01a623d90d0cad0843102f9b8b9fe0d3ff8244593bd817f126582b52dd694",
  ]
}

provider "registry.terraform.io/integrations/github" {
  version     = "4.28.0"
  constraints = "4.28.0"
  hashes = [
    "h1:GMp4fa/6ZxeM8c8O20rZ2jpXXkBqK09oMBtWQ1WwPCo=",
    "h1:PRj9EXEvLgKTmQHKUtzIG28goXJX74aRt0b/4JH6qN8=",
    "h1:vAZrilSL9rq6bXb97dl06QRohtcc0btFQzFF5dHinI0=",
    "zh:125a1decda8a9d4c6d18010f3c66943c868da9e984298c0e2f9dfd240ec660ec",
    "zh:23a4cb334a2fbead38264f434c81e52cb52fb115cbc39537fefc9c22aaecdf35",
    "zh:3cf793b1d0bc30a703315c6ecb6bb2f36d14ed310dec7e300ae4a4a3a470aafe",
    "zh:47cb06845730df19256882272690221db8314199a34012ac7e690e0550ca9404",
    "zh:5d6e76624d60b6298ee47c10cc262adc9f361f4648f40faf81ee3a8d6beaad31",
    "zh:6415a5c6ba5b28f1f410845706cff0390718113f7d987aaa011553b041ba2005",
    "zh:70ce96d7aa424aef47d4b049d39aff036ae6377dacd5c077501eb0f353901cc6",
    "zh:9803fc59cf71ea629308773d429c9ca00985acdcc02d9755fc59900bcf6d1d00",
    "zh:a9a505f208f569ee44a0a6a7c975e3441bb8d61dbf9831c44c3be299e2cf1a21",
    "zh:a9d9a17b0618ea14f9fa49dfc1329b01473a9d708011fca32cd01b474051d169",
    "zh:bce0257085a5d6c9f0e6cdd5a704c50286c5382f840384a2a50c69d8488652bf",
    "zh:d7272bb396e67ff22d7f4628d152fa66610cf7507a4e63d72ef50fde651e39bf",
    "zh:e2aab496c17acb8c2bdd5af9e830e9f91f869d9fc173e6dd65b7475e8baa6f82",
    "zh:ea20984a5386fc4a6856eed58d261c5124fc8ca72bc6ee142c1092036a3c8360",
  ]
}
`,
			want: `
# This file is maintained automatically by "terraform init".
# Manual edits may be lost in future updates.

provider "registry.terraform.io/hashicorp/aws" {
  version     = "5.4.0"
  constraints = "5.4.0"
  hashes = [
    "h1:4eGsUS3r5eApQc19t8woc6d+sQLaOBaCSaK5GyGcWf0=",
    "h1:Jol4lNIzMrREQzUBSveCLX0iQLy7dm0OF+IYY2GKrhY=",
    "h1:ny1YPz2LiHTasDVNh6/HEvh1c9+TN/ftgAHh84bmy1E=",
    "zh:1db5f81089216831bb0fdff9ddc3772efa133397c66ec276bc75b96eec06e23f",
    "zh:26fe5fdf399192b5724d21854fbec650c158f8ee9eb1dc52a50f7da0f2bc07ac",
    "zh:2946d9e333b1efe01588ee9f9771169fd3c3a4a7cb78ed8f91e8b3efd1a73850",
    "zh:36ed69e8d3029332c8a52a70940f714fd579b9fd95f5569cc010ef11162f5bf7",
    "zh:46ba5ad1c3a3ef98c346356cfa4bdd9c2501c661c2513bb92f4413f2482fb24b",
    "zh:46c10aaa9672b54a14b0e0effdd6ecd9b8a539b3bfe273ac54111e7352a7bb4b",
    "zh:47d7f57bcbe4fba2f960ab6c4228c5e9e586be2f233a8baa8962b51a63337179",
    "zh:47e41c198439ba1c4d933f808b6f47e518f8f0aae25ca42abcac97f149121e90",
    "zh:526c5834de71654ee14039cb973322bf5032cb684a2a113b48fb48a0584f46f3",
    "zh:6169316517b95677819ba2904dcea204fb9b55e868348e906af9164104fe7198",
    "zh:7c063ef2b8d69a8db7e8bf0dcd45793ede22b259b30464ed114d330df304cdbb",
    "zh:87c4f2faca636715a08be3121d26b3354415401eab89349077ca9436a0822c23",
    "zh:9b12af85486a96aedd8d7984b0ff811a4b42e3d88dad1a3fb4c0b580d04fa425",
    "zh:b184b8a268f45258edd27d389ca793708f1bc3ee4d6706d154a45e93deaddde1",
    "zh:ba1a998cbf4b639fa3e04b9069f0f5a289662457940726a8a51c81df400aa852",
  ]
}

provider "registry.terraform.io/hashicorp/null" {
  version     = "3.2.1"
  constraints = "3.2.1"
  hashes = [
    "h1:FbGfc+muBsC17Ohy5g806iuI1hQc4SIexpYCrQHQd8w=",
    "h1:tSj1mL6OQ8ILGqR2mDu7OYYYWf+hoir0pf9KAQ8IzO8=",
    "h1:ydA0/SNRVB1o95btfshvYsmxA+jZFRZcvKzZSB+4S1M=",
    "zh:58ed64389620cc7b82f01332e27723856422820cfd302e304b5f6c3436fb9840",
    "zh:62a5cc82c3b2ddef7ef3a6f2fedb7b9b3deff4ab7b414938b08e51d6e8be87cb",
    "zh:63cff4de03af983175a7e37e52d4bd89d990be256b16b5c7f919aff5ad485aa5",
    "zh:74cb22c6700e48486b7cabefa10b33b801dfcab56f1a6ac9b6624531f3d36ea3",
    "zh:78d5eefdd9e494defcb3c68d282b8f96630502cac21d1ea161f53cfe9bb483b3",
    "zh:79e553aff77f1cfa9012a2218b8238dd672ea5e1b2924775ac9ac24d2a75c238",
    "zh:a1e06ddda0b5ac48f7e7c7d59e1ab5a4073bbcf876c73c0299e4610ed53859dc",
    "zh:c37a97090f1a82222925d45d84483b2aa702ef7ab66532af6cbcfb567818b970",
    "zh:e4453fbebf90c53ca3323a92e7ca0f9961427d2f0ce0d2b65523cc04d5d999c2",
    "zh:e80a746921946d8b6761e77305b752ad188da60688cfd2059322875d363be5f5",
    "zh:fbdb892d9822ed0e4cb60f2fedbdbb556e4da0d88d3b942ae963ed6ff091e48f",
    "zh:fca01a623d90d0cad0843102f9b8b9fe0d3ff8244593bd817f126582b52dd694",
  ]
}

provider "registry.terraform.io/integrations/github" {
  version     = "4.28.0"
  constraints = "4.28.0"
  hashes = [
    "h1:GMp4fa/6ZxeM8c8O20rZ2jpXXkBqK09oMBtWQ1WwPCo=",
    "h1:PRj9EXEvLgKTmQHKUtzIG28goXJX74aRt0b/4JH6qN8=",
    "h1:vAZrilSL9rq6bXb97dl06QRohtcc0btFQzFF5dHinI0=",
    "zh:125a1decda8a9d4c6d18010f3c66943c868da9e984298c0e2f9dfd240ec660ec",
    "zh:23a4cb334a2fbead38264f434c81e52cb52fb115cbc39537fefc9c22aaecdf35",
    "zh:3cf793b1d0bc30a703315c6ecb6bb2f36d14ed310dec7e300ae4a4a3a470aafe",
    "zh:47cb06845730df19256882272690221db8314199a34012ac7e690e0550ca9404",
    "zh:5d6e76624d60b6298ee47c10cc262adc9f361f4648f40faf81ee3a8d6beaad31",
    "zh:6415a5c6ba5b28f1f410845706cff0390718113f7d987aaa011553b041ba2005",
    "zh:70ce96d7aa424aef47d4b049d39aff036ae6377dacd5c077501eb0f353901cc6",
    "zh:9803fc59cf71ea629308773d429c9ca00985acdcc02d9755fc59900bcf6d1d00",
    "zh:a9a505f208f569ee44a0a6a7c975e3441bb8d61dbf9831c44c3be299e2cf1a21",
    "zh:a9d9a17b0618ea14f9fa49dfc1329b01473a9d708011fca32cd01b474051d169",
    "zh:bce0257085a5d6c9f0e6cdd5a704c50286c5382f840384a2a50c69d8488652bf",
    "zh:d7272bb396e67ff22d7f4628d152fa66610cf7507a4e63d72ef50fde651e39bf",
    "zh:e2aab496c17acb8c2bdd5af9e830e9f91f869d9fc173e6dd65b7475e8baa6f82",
    "zh:ea20984a5386fc4a6856eed58d261c5124fc8ca72bc6ee142c1092036a3c8360",
  ]
}
`,
			ok: true,
		},
		{
			desc: "update multiple providers",
			src: `
terraform {
  required_version = "1.5.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "5.4.0"
    }

    null = {
      source  = "hashicorp/null"
      version = "3.2.1"
    }

    github = {
      source  = "integrations/github"
      version = "4.28.0"
    }
  }
}
`,
			lockfile: `
# This file is maintained automatically by "terraform init".
# Manual edits may be lost in future updates.

provider "registry.terraform.io/hashicorp/aws" {
  version     = "5.3.0"
  constraints = "5.3.0"
  hashes = [
    "h1:3KbfDs6Sd6pDlVTB9wYCxndEWNmTlShb/rBHnr4/+OE=",
    "h1:89Ara9HnoQzGsFK1nU0fPD8h0SsHJnlVc8mUfOQSAYE=",
    "h1:HKlxtbkaT/YKPtzJLGm8np5JiaPSys/aG6Lbj8QB/5c=",
    "zh:001814dcf6b2329de5e2c9223c4f1e95a0f60d6670046015419053b03b3c0712",
    "zh:3c511a91f53076c3a1117526bee0880b339261f1eb3feecd7854771bfef7890d",
    "zh:3e6c19e048f06051c9296c7a3236946f37431ce0d84f843585c5f3e8504759d3",
    "zh:476a3d918782a479166f33418192b522698e39702e8a0aec823682d3ee3082f1",
    "zh:5dd0d3bff7a7acabeed600dfbbef797e189c4877f65e4b4ed572cb33e454f602",
    "zh:6627f95a41e30c01b7f7c9e3db1cccba056c5257c36cccfaa0898d526211add2",
    "zh:663023a4244cf7f7df2b08ab204922f7902eefe9a7b51a2c2def1a7dafe6f55f",
    "zh:79cb8a22a131b7d2beb331d8443207eed10fdb4b09655048960bd5d59c8bbf3a",
    "zh:8c2275a0954042cfc44843a6045543744e08bd8cad487f0bc9162cf92a9bcdcc",
    "zh:9b12af85486a96aedd8d7984b0ff811a4b42e3d88dad1a3fb4c0b580d04fa425",
    "zh:ad08ae20b9402461af863772a9e4ff5677e14f3fc86d5b148bd4faaaa361f601",
    "zh:b8b7bd15fc1842aeedc2e5eab03b8357cdb2b9fe3e67dd82ae240be3081bf637",
    "zh:bdb3858c4c632aad8d5c4bff063f3afb18de51cec3167b3496d5bc5856915301",
    "zh:f354a433ec8095b06c2701725411ffb73a20ef9b1aa325434e1bb575b5c86d52",
    "zh:f47e1342883d599f4675dcfdeb9707cdfcfaf53c677f93fd5c410580d4dece13",
  ]
}

provider "registry.terraform.io/hashicorp/null" {
  version     = "3.1.1"
  constraints = "3.1.1"
  hashes = [
    "h1:71sNUDvmiJcijsvfXpiLCz0lXIBSsEJjMxljt7hxMhw=",
    "h1:Pctug/s/2Hg5FJqjYcTM0kPyx3AoYK1MpRWO0T9V2ns=",
    "h1:YvH6gTaQzGdNv+SKTZujU1O0bO+Pw6vJHOPhqgN8XNs=",
    "zh:063466f41f1d9fd0dd93722840c1314f046d8760b1812fa67c34de0afcba5597",
    "zh:08c058e367de6debdad35fc24d97131c7cf75103baec8279aba3506a08b53faf",
    "zh:73ce6dff935150d6ddc6ac4a10071e02647d10175c173cfe5dca81f3d13d8afe",
    "zh:78d5eefdd9e494defcb3c68d282b8f96630502cac21d1ea161f53cfe9bb483b3",
    "zh:8fdd792a626413502e68c195f2097352bdc6a0df694f7df350ed784741eb587e",
    "zh:976bbaf268cb497400fd5b3c774d218f3933271864345f18deebe4dcbfcd6afa",
    "zh:b21b78ca581f98f4cdb7a366b03ae9db23a73dfa7df12c533d7c19b68e9e72e5",
    "zh:b7fc0c1615dbdb1d6fd4abb9c7dc7da286631f7ca2299fb9cd4664258ccfbff4",
    "zh:d1efc942b2c44345e0c29bc976594cb7278c38cfb8897b344669eafbc3cddf46",
    "zh:e356c245b3cd9d4789bab010893566acace682d7db877e52d40fc4ca34a50924",
    "zh:ea98802ba92fcfa8cf12cbce2e9e7ebe999afbf8ed47fa45fc847a098d89468b",
    "zh:eff8872458806499889f6927b5d954560f3d74bf20b6043409edf94d26cd906f",
  ]
}

provider "registry.terraform.io/integrations/github" {
  version     = "4.28.0"
  constraints = "4.28.0"
  hashes = [
    "h1:GMp4fa/6ZxeM8c8O20rZ2jpXXkBqK09oMBtWQ1WwPCo=",
    "h1:PRj9EXEvLgKTmQHKUtzIG28goXJX74aRt0b/4JH6qN8=",
    "h1:vAZrilSL9rq6bXb97dl06QRohtcc0btFQzFF5dHinI0=",
    "zh:125a1decda8a9d4c6d18010f3c66943c868da9e984298c0e2f9dfd240ec660ec",
    "zh:23a4cb334a2fbead38264f434c81e52cb52fb115cbc39537fefc9c22aaecdf35",
    "zh:3cf793b1d0bc30a703315c6ecb6bb2f36d14ed310dec7e300ae4a4a3a470aafe",
    "zh:47cb06845730df19256882272690221db8314199a34012ac7e690e0550ca9404",
    "zh:5d6e76624d60b6298ee47c10cc262adc9f361f4648f40faf81ee3a8d6beaad31",
    "zh:6415a5c6ba5b28f1f410845706cff0390718113f7d987aaa011553b041ba2005",
    "zh:70ce96d7aa424aef47d4b049d39aff036ae6377dacd5c077501eb0f353901cc6",
    "zh:9803fc59cf71ea629308773d429c9ca00985acdcc02d9755fc59900bcf6d1d00",
    "zh:a9a505f208f569ee44a0a6a7c975e3441bb8d61dbf9831c44c3be299e2cf1a21",
    "zh:a9d9a17b0618ea14f9fa49dfc1329b01473a9d708011fca32cd01b474051d169",
    "zh:bce0257085a5d6c9f0e6cdd5a704c50286c5382f840384a2a50c69d8488652bf",
    "zh:d7272bb396e67ff22d7f4628d152fa66610cf7507a4e63d72ef50fde651e39bf",
    "zh:e2aab496c17acb8c2bdd5af9e830e9f91f869d9fc173e6dd65b7475e8baa6f82",
    "zh:ea20984a5386fc4a6856eed58d261c5124fc8ca72bc6ee142c1092036a3c8360",
  ]
}
`,
			want: `
# This file is maintained automatically by "terraform init".
# Manual edits may be lost in future updates.

provider "registry.terraform.io/hashicorp/aws" {
  version     = "5.4.0"
  constraints = "5.4.0"
  hashes = [
    "h1:4eGsUS3r5eApQc19t8woc6d+sQLaOBaCSaK5GyGcWf0=",
    "h1:Jol4lNIzMrREQzUBSveCLX0iQLy7dm0OF+IYY2GKrhY=",
    "h1:ny1YPz2LiHTasDVNh6/HEvh1c9+TN/ftgAHh84bmy1E=",
    "zh:1db5f81089216831bb0fdff9ddc3772efa133397c66ec276bc75b96eec06e23f",
    "zh:26fe5fdf399192b5724d21854fbec650c158f8ee9eb1dc52a50f7da0f2bc07ac",
    "zh:2946d9e333b1efe01588ee9f9771169fd3c3a4a7cb78ed8f91e8b3efd1a73850",
    "zh:36ed69e8d3029332c8a52a70940f714fd579b9fd95f5569cc010ef11162f5bf7",
    "zh:46ba5ad1c3a3ef98c346356cfa4bdd9c2501c661c2513bb92f4413f2482fb24b",
    "zh:46c10aaa9672b54a14b0e0effdd6ecd9b8a539b3bfe273ac54111e7352a7bb4b",
    "zh:47d7f57bcbe4fba2f960ab6c4228c5e9e586be2f233a8baa8962b51a63337179",
    "zh:47e41c198439ba1c4d933f808b6f47e518f8f0aae25ca42abcac97f149121e90",
    "zh:526c5834de71654ee14039cb973322bf5032cb684a2a113b48fb48a0584f46f3",
    "zh:6169316517b95677819ba2904dcea204fb9b55e868348e906af9164104fe7198",
    "zh:7c063ef2b8d69a8db7e8bf0dcd45793ede22b259b30464ed114d330df304cdbb",
    "zh:87c4f2faca636715a08be3121d26b3354415401eab89349077ca9436a0822c23",
    "zh:9b12af85486a96aedd8d7984b0ff811a4b42e3d88dad1a3fb4c0b580d04fa425",
    "zh:b184b8a268f45258edd27d389ca793708f1bc3ee4d6706d154a45e93deaddde1",
    "zh:ba1a998cbf4b639fa3e04b9069f0f5a289662457940726a8a51c81df400aa852",
  ]
}

provider "registry.terraform.io/hashicorp/null" {
  version     = "3.2.1"
  constraints = "3.2.1"
  hashes = [
    "h1:FbGfc+muBsC17Ohy5g806iuI1hQc4SIexpYCrQHQd8w=",
    "h1:tSj1mL6OQ8ILGqR2mDu7OYYYWf+hoir0pf9KAQ8IzO8=",
    "h1:ydA0/SNRVB1o95btfshvYsmxA+jZFRZcvKzZSB+4S1M=",
    "zh:58ed64389620cc7b82f01332e27723856422820cfd302e304b5f6c3436fb9840",
    "zh:62a5cc82c3b2ddef7ef3a6f2fedb7b9b3deff4ab7b414938b08e51d6e8be87cb",
    "zh:63cff4de03af983175a7e37e52d4bd89d990be256b16b5c7f919aff5ad485aa5",
    "zh:74cb22c6700e48486b7cabefa10b33b801dfcab56f1a6ac9b6624531f3d36ea3",
    "zh:78d5eefdd9e494defcb3c68d282b8f96630502cac21d1ea161f53cfe9bb483b3",
    "zh:79e553aff77f1cfa9012a2218b8238dd672ea5e1b2924775ac9ac24d2a75c238",
    "zh:a1e06ddda0b5ac48f7e7c7d59e1ab5a4073bbcf876c73c0299e4610ed53859dc",
    "zh:c37a97090f1a82222925d45d84483b2aa702ef7ab66532af6cbcfb567818b970",
    "zh:e4453fbebf90c53ca3323a92e7ca0f9961427d2f0ce0d2b65523cc04d5d999c2",
    "zh:e80a746921946d8b6761e77305b752ad188da60688cfd2059322875d363be5f5",
    "zh:fbdb892d9822ed0e4cb60f2fedbdbb556e4da0d88d3b942ae963ed6ff091e48f",
    "zh:fca01a623d90d0cad0843102f9b8b9fe0d3ff8244593bd817f126582b52dd694",
  ]
}

provider "registry.terraform.io/integrations/github" {
  version     = "4.28.0"
  constraints = "4.28.0"
  hashes = [
    "h1:GMp4fa/6ZxeM8c8O20rZ2jpXXkBqK09oMBtWQ1WwPCo=",
    "h1:PRj9EXEvLgKTmQHKUtzIG28goXJX74aRt0b/4JH6qN8=",
    "h1:vAZrilSL9rq6bXb97dl06QRohtcc0btFQzFF5dHinI0=",
    "zh:125a1decda8a9d4c6d18010f3c66943c868da9e984298c0e2f9dfd240ec660ec",
    "zh:23a4cb334a2fbead38264f434c81e52cb52fb115cbc39537fefc9c22aaecdf35",
    "zh:3cf793b1d0bc30a703315c6ecb6bb2f36d14ed310dec7e300ae4a4a3a470aafe",
    "zh:47cb06845730df19256882272690221db8314199a34012ac7e690e0550ca9404",
    "zh:5d6e76624d60b6298ee47c10cc262adc9f361f4648f40faf81ee3a8d6beaad31",
    "zh:6415a5c6ba5b28f1f410845706cff0390718113f7d987aaa011553b041ba2005",
    "zh:70ce96d7aa424aef47d4b049d39aff036ae6377dacd5c077501eb0f353901cc6",
    "zh:9803fc59cf71ea629308773d429c9ca00985acdcc02d9755fc59900bcf6d1d00",
    "zh:a9a505f208f569ee44a0a6a7c975e3441bb8d61dbf9831c44c3be299e2cf1a21",
    "zh:a9d9a17b0618ea14f9fa49dfc1329b01473a9d708011fca32cd01b474051d169",
    "zh:bce0257085a5d6c9f0e6cdd5a704c50286c5382f840384a2a50c69d8488652bf",
    "zh:d7272bb396e67ff22d7f4628d152fa66610cf7507a4e63d72ef50fde651e39bf",
    "zh:e2aab496c17acb8c2bdd5af9e830e9f91f869d9fc173e6dd65b7475e8baa6f82",
    "zh:ea20984a5386fc4a6856eed58d261c5124fc8ca72bc6ee142c1092036a3c8360",
  ]
}
`,
			ok: true,
		},
		{
			desc: "create missing providers",
			src: `
terraform {
  required_version = "1.5.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "5.4.0"
    }

    null = {
      source  = "hashicorp/null"
      version = "3.2.1"
    }

    github = {
      source  = "integrations/github"
      version = "4.28.0"
    }
  }
}
`,
			lockfile: `
# This file is maintained automatically by "terraform init".
# Manual edits may be lost in future updates.

provider "registry.terraform.io/integrations/github" {
  version     = "4.28.0"
  constraints = "4.28.0"
  hashes = [
    "h1:GMp4fa/6ZxeM8c8O20rZ2jpXXkBqK09oMBtWQ1WwPCo=",
    "h1:PRj9EXEvLgKTmQHKUtzIG28goXJX74aRt0b/4JH6qN8=",
    "h1:vAZrilSL9rq6bXb97dl06QRohtcc0btFQzFF5dHinI0=",
    "zh:125a1decda8a9d4c6d18010f3c66943c868da9e984298c0e2f9dfd240ec660ec",
    "zh:23a4cb334a2fbead38264f434c81e52cb52fb115cbc39537fefc9c22aaecdf35",
    "zh:3cf793b1d0bc30a703315c6ecb6bb2f36d14ed310dec7e300ae4a4a3a470aafe",
    "zh:47cb06845730df19256882272690221db8314199a34012ac7e690e0550ca9404",
    "zh:5d6e76624d60b6298ee47c10cc262adc9f361f4648f40faf81ee3a8d6beaad31",
    "zh:6415a5c6ba5b28f1f410845706cff0390718113f7d987aaa011553b041ba2005",
    "zh:70ce96d7aa424aef47d4b049d39aff036ae6377dacd5c077501eb0f353901cc6",
    "zh:9803fc59cf71ea629308773d429c9ca00985acdcc02d9755fc59900bcf6d1d00",
    "zh:a9a505f208f569ee44a0a6a7c975e3441bb8d61dbf9831c44c3be299e2cf1a21",
    "zh:a9d9a17b0618ea14f9fa49dfc1329b01473a9d708011fca32cd01b474051d169",
    "zh:bce0257085a5d6c9f0e6cdd5a704c50286c5382f840384a2a50c69d8488652bf",
    "zh:d7272bb396e67ff22d7f4628d152fa66610cf7507a4e63d72ef50fde651e39bf",
    "zh:e2aab496c17acb8c2bdd5af9e830e9f91f869d9fc173e6dd65b7475e8baa6f82",
    "zh:ea20984a5386fc4a6856eed58d261c5124fc8ca72bc6ee142c1092036a3c8360",
  ]
}
`,
			want: `
# This file is maintained automatically by "terraform init".
# Manual edits may be lost in future updates.

provider "registry.terraform.io/integrations/github" {
  version     = "4.28.0"
  constraints = "4.28.0"
  hashes = [
    "h1:GMp4fa/6ZxeM8c8O20rZ2jpXXkBqK09oMBtWQ1WwPCo=",
    "h1:PRj9EXEvLgKTmQHKUtzIG28goXJX74aRt0b/4JH6qN8=",
    "h1:vAZrilSL9rq6bXb97dl06QRohtcc0btFQzFF5dHinI0=",
    "zh:125a1decda8a9d4c6d18010f3c66943c868da9e984298c0e2f9dfd240ec660ec",
    "zh:23a4cb334a2fbead38264f434c81e52cb52fb115cbc39537fefc9c22aaecdf35",
    "zh:3cf793b1d0bc30a703315c6ecb6bb2f36d14ed310dec7e300ae4a4a3a470aafe",
    "zh:47cb06845730df19256882272690221db8314199a34012ac7e690e0550ca9404",
    "zh:5d6e76624d60b6298ee47c10cc262adc9f361f4648f40faf81ee3a8d6beaad31",
    "zh:6415a5c6ba5b28f1f410845706cff0390718113f7d987aaa011553b041ba2005",
    "zh:70ce96d7aa424aef47d4b049d39aff036ae6377dacd5c077501eb0f353901cc6",
    "zh:9803fc59cf71ea629308773d429c9ca00985acdcc02d9755fc59900bcf6d1d00",
    "zh:a9a505f208f569ee44a0a6a7c975e3441bb8d61dbf9831c44c3be299e2cf1a21",
    "zh:a9d9a17b0618ea14f9fa49dfc1329b01473a9d708011fca32cd01b474051d169",
    "zh:bce0257085a5d6c9f0e6cdd5a704c50286c5382f840384a2a50c69d8488652bf",
    "zh:d7272bb396e67ff22d7f4628d152fa66610cf7507a4e63d72ef50fde651e39bf",
    "zh:e2aab496c17acb8c2bdd5af9e830e9f91f869d9fc173e6dd65b7475e8baa6f82",
    "zh:ea20984a5386fc4a6856eed58d261c5124fc8ca72bc6ee142c1092036a3c8360",
  ]
}

provider "registry.terraform.io/hashicorp/aws" {
  version     = "5.4.0"
  constraints = "5.4.0"
  hashes = [
    "h1:4eGsUS3r5eApQc19t8woc6d+sQLaOBaCSaK5GyGcWf0=",
    "h1:Jol4lNIzMrREQzUBSveCLX0iQLy7dm0OF+IYY2GKrhY=",
    "h1:ny1YPz2LiHTasDVNh6/HEvh1c9+TN/ftgAHh84bmy1E=",
    "zh:1db5f81089216831bb0fdff9ddc3772efa133397c66ec276bc75b96eec06e23f",
    "zh:26fe5fdf399192b5724d21854fbec650c158f8ee9eb1dc52a50f7da0f2bc07ac",
    "zh:2946d9e333b1efe01588ee9f9771169fd3c3a4a7cb78ed8f91e8b3efd1a73850",
    "zh:36ed69e8d3029332c8a52a70940f714fd579b9fd95f5569cc010ef11162f5bf7",
    "zh:46ba5ad1c3a3ef98c346356cfa4bdd9c2501c661c2513bb92f4413f2482fb24b",
    "zh:46c10aaa9672b54a14b0e0effdd6ecd9b8a539b3bfe273ac54111e7352a7bb4b",
    "zh:47d7f57bcbe4fba2f960ab6c4228c5e9e586be2f233a8baa8962b51a63337179",
    "zh:47e41c198439ba1c4d933f808b6f47e518f8f0aae25ca42abcac97f149121e90",
    "zh:526c5834de71654ee14039cb973322bf5032cb684a2a113b48fb48a0584f46f3",
    "zh:6169316517b95677819ba2904dcea204fb9b55e868348e906af9164104fe7198",
    "zh:7c063ef2b8d69a8db7e8bf0dcd45793ede22b259b30464ed114d330df304cdbb",
    "zh:87c4f2faca636715a08be3121d26b3354415401eab89349077ca9436a0822c23",
    "zh:9b12af85486a96aedd8d7984b0ff811a4b42e3d88dad1a3fb4c0b580d04fa425",
    "zh:b184b8a268f45258edd27d389ca793708f1bc3ee4d6706d154a45e93deaddde1",
    "zh:ba1a998cbf4b639fa3e04b9069f0f5a289662457940726a8a51c81df400aa852",
  ]
}

provider "registry.terraform.io/hashicorp/null" {
  version     = "3.2.1"
  constraints = "3.2.1"
  hashes = [
    "h1:FbGfc+muBsC17Ohy5g806iuI1hQc4SIexpYCrQHQd8w=",
    "h1:tSj1mL6OQ8ILGqR2mDu7OYYYWf+hoir0pf9KAQ8IzO8=",
    "h1:ydA0/SNRVB1o95btfshvYsmxA+jZFRZcvKzZSB+4S1M=",
    "zh:58ed64389620cc7b82f01332e27723856422820cfd302e304b5f6c3436fb9840",
    "zh:62a5cc82c3b2ddef7ef3a6f2fedb7b9b3deff4ab7b414938b08e51d6e8be87cb",
    "zh:63cff4de03af983175a7e37e52d4bd89d990be256b16b5c7f919aff5ad485aa5",
    "zh:74cb22c6700e48486b7cabefa10b33b801dfcab56f1a6ac9b6624531f3d36ea3",
    "zh:78d5eefdd9e494defcb3c68d282b8f96630502cac21d1ea161f53cfe9bb483b3",
    "zh:79e553aff77f1cfa9012a2218b8238dd672ea5e1b2924775ac9ac24d2a75c238",
    "zh:a1e06ddda0b5ac48f7e7c7d59e1ab5a4073bbcf876c73c0299e4610ed53859dc",
    "zh:c37a97090f1a82222925d45d84483b2aa702ef7ab66532af6cbcfb567818b970",
    "zh:e4453fbebf90c53ca3323a92e7ca0f9961427d2f0ce0d2b65523cc04d5d999c2",
    "zh:e80a746921946d8b6761e77305b752ad188da60688cfd2059322875d363be5f5",
    "zh:fbdb892d9822ed0e4cb60f2fedbdbb556e4da0d88d3b942ae963ed6ff091e48f",
    "zh:fca01a623d90d0cad0843102f9b8b9fe0d3ff8244593bd817f126582b52dd694",
  ]
}
`,
			ok: true,
		},
		{
			desc: "create from empty",
			src: `
terraform {
  required_version = "1.5.0"

  required_providers {
    null = {
      source  = "hashicorp/null"
      version = "3.2.1"
    }
  }
}
`,
			lockfile: ``,
			want: `
provider "registry.terraform.io/hashicorp/null" {
  version     = "3.2.1"
  constraints = "3.2.1"
  hashes = [
    "h1:FbGfc+muBsC17Ohy5g806iuI1hQc4SIexpYCrQHQd8w=",
    "h1:tSj1mL6OQ8ILGqR2mDu7OYYYWf+hoir0pf9KAQ8IzO8=",
    "h1:ydA0/SNRVB1o95btfshvYsmxA+jZFRZcvKzZSB+4S1M=",
    "zh:58ed64389620cc7b82f01332e27723856422820cfd302e304b5f6c3436fb9840",
    "zh:62a5cc82c3b2ddef7ef3a6f2fedb7b9b3deff4ab7b414938b08e51d6e8be87cb",
    "zh:63cff4de03af983175a7e37e52d4bd89d990be256b16b5c7f919aff5ad485aa5",
    "zh:74cb22c6700e48486b7cabefa10b33b801dfcab56f1a6ac9b6624531f3d36ea3",
    "zh:78d5eefdd9e494defcb3c68d282b8f96630502cac21d1ea161f53cfe9bb483b3",
    "zh:79e553aff77f1cfa9012a2218b8238dd672ea5e1b2924775ac9ac24d2a75c238",
    "zh:a1e06ddda0b5ac48f7e7c7d59e1ab5a4073bbcf876c73c0299e4610ed53859dc",
    "zh:c37a97090f1a82222925d45d84483b2aa702ef7ab66532af6cbcfb567818b970",
    "zh:e4453fbebf90c53ca3323a92e7ca0f9961427d2f0ce0d2b65523cc04d5d999c2",
    "zh:e80a746921946d8b6761e77305b752ad188da60688cfd2059322875d363be5f5",
    "zh:fbdb892d9822ed0e4cb60f2fedbdbb556e4da0d88d3b942ae963ed6ff091e48f",
    "zh:fca01a623d90d0cad0843102f9b8b9fe0d3ff8244593bd817f126582b52dd694",
  ]
}
`,
			ok: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			dirname := "test"
			err := fs.MkdirAll(dirname, os.ModePerm)
			if err != nil {
				t.Fatalf("failed to create dir: %s", err)
			}

			err = afero.WriteFile(fs, filepath.Join(dirname, "main.tf"), []byte(tc.src), 0644)
			if err != nil {
				t.Fatalf("failed to write file: %s", err)
			}

			err = afero.WriteFile(fs, filepath.Join(dirname, ".terraform.lock.hcl"), []byte(tc.lockfile), 0644)
			if err != nil {
				t.Fatalf("failed to write file: %s", err)
			}

			o := Option{
				updateType: "lock",
				platforms:  platforms,
			}
			gc, err := NewGlobalContext(fs, o)
			if err != nil {
				t.Fatalf("failed to new global context: %s", err)
			}

			// Create a mock index for testing
			mockIndex := lock.NewMockIndex(pvs)

			// Create a LockUpdater with empty tfregistryConfig
			u, err := NewLockUpdater(platforms, tfregistry.Config{})

			// Replace the index with our mock for testing
			lu := u.(*LockUpdater)
			lu.index = mockIndex
			if err != nil {
				t.Fatalf("failed to new LockUpdater: %s", err)
			}

			f, diags := hclwrite.ParseConfig([]byte(tc.lockfile), ".terraform.lock.hcl", hcl.Pos{Line: 1, Column: 1})
			if diags.HasErrors() {
				t.Fatalf("unexpected diagnostics: %s", diags)
			}

			mc, err := NewModuleContext(dirname, gc)
			if err != nil {
				t.Fatalf("failed to new module context: %s", err)
			}

			err = u.Update(context.Background(), mc, ".terraform.lock.hcl", f)
			if tc.ok && err != nil {
				t.Errorf("failed to call Update: err = %s", err)
			}

			got := string(hclwrite.Format(f.BuildTokens(nil).Bytes()))

			if !tc.ok && err == nil {
				t.Errorf("expect to fail, but success: got = %s", got)
			}

			if diff := cmp.Diff(got, tc.want); diff != "" {
				t.Errorf("got: %s, want = %s, diff = %s", got, tc.want, diff)
			}
		})
	}
}

// constraintTransport keeps registry requests offline while exercising the real client.
type constraintTransport func(*http.Request) (*http.Response, error)

func (f constraintTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type constraintFixture struct {
	updater           Updater
	config            tfregistry.Config
	checksumCalls     []string
	checksumPlatforms []string
	checksumFailure   string
	catalogs          map[string][]tfregistry.ProviderVersion
	requests          []string
	failures          int
	badJSON           bool
	transportError    bool
}

func newConstraintFixture(t *testing.T, platforms []string) *constraintFixture {
	t.Helper()
	x := &constraintFixture{catalogs: map[string][]tfregistry.ProviderVersion{}}
	config := tfregistry.Config{BaseURL: "https://mirror.example/prefix/", HTTPClient: &http.Client{Transport: constraintTransport(func(r *http.Request) (*http.Response, error) {
		if !strings.HasSuffix(r.URL.Path, "/versions") {
			return x.checksumResponse(t, r)
		}
		x.requests = append(x.requests, r.URL.String())
		if r.URL.Host != "mirror.example" || !strings.HasPrefix(r.URL.Path, "/prefix/v1/providers/") {
			return nil, fmt.Errorf("wrong endpoint: %s", r.URL)
		}
		if x.transportError {
			return nil, fmt.Errorf("transport unavailable")
		}
		status := 200
		if x.failures > 0 {
			x.failures--
			status = 503
		}
		key := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/prefix/v1/providers/"), "/versions")
		data, err := json.Marshal(tfregistry.ListProviderVersionsResponse{Versions: x.catalogs[key]})
		if err != nil {
			return nil, err
		}
		if x.badJSON {
			data = []byte("{broken")
		}
		return &http.Response{StatusCode: status, Body: io.NopCloser(bytes.NewReader(data)), Header: make(http.Header)}, nil
	})}}
	u, err := NewLockUpdater(platforms, config)
	if err != nil {
		t.Fatal(err)
	}
	x.updater = u
	x.config = config
	return x
}
func (x *constraintFixture) checksumResponse(t *testing.T, r *http.Request) (*http.Response, error) {
	t.Helper()
	if r.URL.Host != "mirror.example" || !strings.HasPrefix(r.URL.Path, "/prefix/") {
		return nil, fmt.Errorf("wrong checksum endpoint: %s", r.URL)
	}
	response := func(data []byte) *http.Response {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(data)), Header: make(http.Header)}
	}
	var archive bytes.Buffer
	zw := zip.NewWriter(&archive)
	w, err := zw.Create("provider")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("offline provider fixture")); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	sum := fmt.Sprintf("%x", sha256.Sum256(archive.Bytes()))
	switch {
	case strings.HasSuffix(r.URL.Path, "/package.zip"):
		return response(archive.Bytes()), nil
	case strings.HasSuffix(r.URL.Path, "/SHA256SUMS"):
		return response([]byte(sum + "  provider.zip\n")), nil
	case strings.Contains(r.URL.Path, "/download/"):
		parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/prefix/v1/providers/"), "/")
		if len(parts) != 6 || parts[3] != "download" {
			return nil, fmt.Errorf("invalid metadata URL: %s", r.URL)
		}
		address, version, platform := parts[0]+"/"+parts[1], parts[2], parts[4]+"_"+parts[5]
		x.checksumCalls = append(x.checksumCalls, address+"@"+version)
		x.checksumPlatforms = append(x.checksumPlatforms, platform)
		if x.checksumFailure != "" && strings.Contains(address, x.checksumFailure) {
			return nil, fmt.Errorf("checksum unavailable")
		}
		data := []byte(fmt.Sprintf(`{"filename":"provider.zip","download_url":"https://mirror.example/prefix/package.zip","shasums_url":"https://mirror.example/prefix/SHA256SUMS","shasum":%q,"packages":{"linux_amd64":{"hashes":[%q,%q]}}}`, sum, "h1:"+version, "zh:"+version))
		return response(data), nil
	default:
		return nil, fmt.Errorf("unexpected checksum URL: %s", r.URL)
	}
}

func constraintCatalog(versions ...string) []tfregistry.ProviderVersion {
	out := []tfregistry.ProviderVersion{}
	for _, v := range versions {
		out = append(out, tfregistry.ProviderVersion{Version: v})
	}
	return out
}
func constraintRequirement(source string, expressions ...string) *tfconfig.ProviderRequirement {
	return &tfconfig.ProviderRequirement{Source: source, VersionConstraints: expressions}
}
func constraintAST(t *testing.T, input string) *hclwrite.File {
	t.Helper()
	f, d := hclwrite.ParseConfig([]byte(input), ".terraform.lock.hcl", hcl.Pos{Line: 1, Column: 1})
	if d.HasErrors() {
		t.Fatal(d)
	}
	return f
}
func (x *constraintFixture) apply(t *testing.T, f *hclwrite.File, requirements map[string]*tfconfig.ProviderRequirement) error {
	t.Helper()
	mc := &ModuleContext{dir: "workspace/module", requiredProviders: requirements}
	return x.updater.Update(context.Background(), mc, ".terraform.lock.hcl", f)
}
func assertErrorContext(t *testing.T, err error, identities ...string) {
	t.Helper()
	if err == nil {
		t.Fatal("expected contextual error, got success")
	}
	for _, identity := range identities {
		if !strings.Contains(err.Error(), identity) {
			t.Errorf("error %q does not identify %q", err, identity)
		}
	}
}

// Provider identities may use a declared name, raw source or canonical address.
// Only the module identity is fixed; human-readable error wording is not.
func assertProviderErrorContext(t *testing.T, err error, module string, names ...string) {
	t.Helper()
	assertErrorContext(t, err, module)
	words := strings.FieldsFunc(err.Error(), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r) && !strings.ContainsRune("_./-", r)
	})
	for _, word := range words {
		for _, name := range names {
			if word == name || strings.HasSuffix(word, "/"+name) {
				return
			}
		}
	}
	t.Errorf("error %q does not identify provider %v", err, names)
}
func assertConstraintEntry(t *testing.T, f *hclwrite.File, source, v, clauses string) {
	t.Helper()
	b := f.Body().FirstMatchingBlock("provider", []string{source})
	if b == nil {
		t.Fatalf("missing provider %s: %s", source, f.Bytes())
	}
	for name, want := range map[string]string{"version": v, "constraints": clauses} {
		a := b.Body().GetAttribute(name)
		if a == nil {
			t.Fatalf("missing %s", name)
		}
		value := getAttributeValueAsUnquotedString(a)
		if value != want {
			t.Errorf("%s = %q, want %q", name, value, want)
		}
	}
}

func TestConstraintLockSelection(t *testing.T) {
	cases := []struct {
		name         string
		expressions  []string
		catalog      []string
		want, record string
	}{
		{"pessimistic_minor", []string{"~> 2.1"}, []string{"3.0", "2.9", "2.1", "2.10"}, "2.10.0", "~> 2.1"},
		{"pessimistic_patch", []string{"~> 2.1.2"}, []string{"2.2", "2.1.9", "2.1.1"}, "2.1.9", "~> 2.1.2"},
		{"greater", []string{"> 2.1"}, []string{"2.1", "2.1.1", "1.9"}, "2.1.1", "> 2.1"},
		{"greater_equal", []string{">= 2.1"}, []string{"2.0.9", "2.1"}, "2.1.0", ">= 2.1"},
		{"less", []string{"< 2.1"}, []string{"2.1", "2.0.9", "2.2"}, "2.0.9", "< 2.1"},
		{"less_equal", []string{"<= 2.1"}, []string{"2.1.1", "2.1", "2.0.9"}, "2.1.0", "<= 2.1"},
		{"exclude", []string{">= 2, != 2.9"}, []string{"2.9", "2.8", "1.9"}, "2.8.0", "!= 2.9, >= 2"},
		{"conjunction", []string{">= 2, < 3", "!= 2.9"}, []string{"3", "2.9", "2.8", "1.9"}, "2.8.0", "!= 2.9, < 3, >= 2"},
		{"normalize_and_deduplicate", []string{" >= 2 , < 3 ", "< 3", ">= 2"}, []string{"v2.7", "3"}, "2.7.0", "< 3, >= 2"},
		{"spelling", []string{">=2 , <3", ">= 2"}, []string{"2.3"}, "2.3.0", "<3, >= 2, >=2"},
		{"ignore_malformed", []string{">= 1"}, []string{"bad", "2.5", "", "worse", "2.4"}, "2.5.0", ">= 1"},
		{"stable_excludes_prerelease", []string{">= 1"}, []string{"9.0.0-beta.2", "2.5"}, "2.5.0", ">= 1"},
		{"prerelease_range", []string{">= 2.0.0-beta.1, < 2.0.0-beta.9"}, []string{"2.0.0-beta.0", "2.0.0-beta.8", "2.0.0-beta.9", "2.1.0-beta.3"}, "2.0.0-beta.8", "< 2.0.0-beta.9, >= 2.0.0-beta.1"},
		{"metadata_tie", []string{">= 1"}, []string{"2.0.0+z", "2.0.0+a", "2.0.0"}, "2.0.0+z", ">= 1"},
		{"second_metadata_tie", []string{">= 4"}, []string{"4.7.0+9", "4.7.0+10", "4.7.0"}, "4.7.0+9", ">= 4"},
		{"equality_pin", []string{"= 2.3"}, nil, "2.3.0", "= 2.3"},
		{"pin_and_range", []string{"2.3", ">= 2", "< 3"}, nil, "2.3.0", "2.3, < 3, >= 2"},
		{"equivalent_pins", []string{"=2.3.0", "v2.3", "2.3"}, nil, "2.3.0", "2.3, =2.3.0, v2.3"},
		{"prerelease_pin", []string{"= 2.0.0-beta.3"}, nil, "2.0.0-beta.3", "= 2.0.0-beta.3"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, reverse := range []bool{false, true} {
				x := newConstraintFixture(t, nil)
				catalog := append([]string(nil), tc.catalog...)
				if reverse {
					slices.Reverse(catalog)
				}
				x.catalogs["acme/widget"] = constraintCatalog(catalog...)
				f := constraintAST(t, "")
				err := x.apply(t, f, map[string]*tfconfig.ProviderRequirement{"widget": constraintRequirement("acme/widget", tc.expressions...)})
				if err != nil {
					t.Fatal(err)
				}
				assertConstraintEntry(t, f, "mirror.example/acme/widget", tc.want, tc.record)
				if tc.catalog == nil && len(x.requests) != 0 {
					t.Fatalf("exact pin listed catalog: %v", x.requests)
				}
				if len(x.checksumCalls) != 1 || !strings.HasSuffix(x.checksumCalls[0], "@"+tc.want) {
					t.Fatalf("checksum calls: %v", x.checksumCalls)
				}
				b := f.Body().FirstMatchingBlock("provider", []string{"mirror.example/acme/widget"})
				hashes := string(b.Body().GetAttribute("hashes").BuildTokens(nil).Bytes())
				if !strings.Contains(hashes, "h1:"+tc.want) || !strings.Contains(hashes, "zh:"+tc.want) {
					t.Fatalf("wrong hashes: %s", hashes)
				}
			}
		})
	}
}

func TestConstraintLockErrors(t *testing.T) {
	cases := []struct {
		name        string
		expressions []string
		catalog     []string
		kind        string
	}{
		{"empty_expression", []string{""}, nil, "invalid"},
		{"whitespace", []string{"  "}, nil, "invalid"},
		{"bad_operator", []string{"=> 2"}, nil, "invalid"},
		{"trailing_comma", []string{">= 2,"}, nil, "invalid"},
		{"bad_after_pin", []string{"2.3", "nonsense"}, nil, "invalid"},
		{"conflicting_pins", []string{"2.3", "= 2.4"}, nil, "conflict"},
		{"pin_outside_range", []string{"2.3", "> 3"}, nil, "conflict"},
		{"excluded_pin", []string{"2.3", "!= 2.3"}, nil, "conflict"},
		{"empty_catalog", []string{">= 2"}, nil, "no_match"},
		{"strict_lower_endpoint", []string{"> 2.1"}, []string{"2.1", "2.0.9"}, "no_match"},
		{"malformed_only_catalog", []string{">= 2"}, []string{"bad", ""}, "no_match"},
		{"outside_catalog", []string{">= 2"}, []string{"1.9"}, "no_match"},
		{"disjoint", []string{">= 3", "< 2"}, []string{"1", "2", "3"}, "no_match"},
		{"http", []string{">= 2"}, nil, "http"},
		{"json", []string{">= 2"}, nil, "json"},
		{"transport", []string{">= 2"}, nil, "transport"},
		{"checksum", []string{"= 2.3"}, nil, "checksum"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, input := range []string{"", "provider \"mirror.example/aaa/first\" {\n version=\"1.0.0\"\n constraints=\"old\"\n hashes=[\"h1:old\"]\n}\n"} {
				x := newConstraintFixture(t, nil)
				x.catalogs["acme/widget"] = constraintCatalog(tc.catalog...)
				switch tc.kind {
				case "http":
					x.failures = 1
				case "json":
					x.badJSON = true
				case "transport":
					x.transportError = true
				case "checksum":
					x.checksumFailure = "acme/widget"
				}
				f := constraintAST(t, input)
				before := string(f.Bytes())
				err := x.apply(t, f, map[string]*tfconfig.ProviderRequirement{"first": constraintRequirement("aaa/first", "= 1"), "widget": constraintRequirement("acme/widget", tc.expressions...)})
				if err == nil {
					t.Fatal("expected resolution/checksum error")
				}
				assertProviderErrorContext(t, err, "workspace/module", "widget", "p")
				if string(f.Bytes()) != before {
					t.Fatalf("failure mutated AST: %s", f.Bytes())
				}
			}
		})
	}
}

func TestConstraintLockPlatforms(t *testing.T) {
	linux := tfregistry.ProviderPlatform{OS: "linux", Arch: "amd64"}
	darwin := tfregistry.ProviderPlatform{OS: "darwin", Arch: "arm64"}
	both := []tfregistry.ProviderPlatform{linux, darwin}
	cases := []struct {
		name      string
		platforms []string
		catalog   []tfregistry.ProviderVersion
		want      string
	}{
		{"all_targets", []string{"linux_amd64", "darwin_arm64"}, []tfregistry.ProviderVersion{{Version: "3", Platforms: []tfregistry.ProviderPlatform{linux}}, {Version: "2", Platforms: both}}, "2.0.0"},
		{"duplicate_normalized_fragments", []string{"linux_amd64", "darwin_arm64"}, []tfregistry.ProviderVersion{{Version: "3", Platforms: []tfregistry.ProviderPlatform{linux, linux}}, {Version: "v3.0.0", Platforms: []tfregistry.ProviderPlatform{darwin}}, {Version: "2", Platforms: both}}, "3.0.0"},
		{"duplicate_requested_platforms", []string{"linux_amd64", "linux_amd64"}, []tfregistry.ProviderVersion{{Version: "3", Platforms: []tfregistry.ProviderPlatform{linux, linux}}}, "3.0.0"},
		{"missing_metadata", []string{"linux_amd64"}, []tfregistry.ProviderVersion{{Version: "3"}, {Version: "2", Platforms: both}}, "2.0.0"},
		{"no_targets", nil, []tfregistry.ProviderVersion{{Version: "3"}, {Version: "2", Platforms: both}}, "3.0.0"},
		{"no_compatible", []string{"linux_amd64", "darwin_arm64"}, []tfregistry.ProviderVersion{{Version: "3", Platforms: []tfregistry.ProviderPlatform{linux}}}, ""},
		{"no_cross_version_union", []string{"linux_amd64", "darwin_arm64"}, []tfregistry.ProviderVersion{{Version: "3", Platforms: []tfregistry.ProviderPlatform{linux}}, {Version: "2", Platforms: []tfregistry.ProviderPlatform{darwin}}}, ""},
		{"no_cross_metadata_union", []string{"linux_amd64", "darwin_arm64"}, []tfregistry.ProviderVersion{{Version: "3.0.0+a", Platforms: []tfregistry.ProviderPlatform{linux}}, {Version: "3.0.0+b", Platforms: []tfregistry.ProviderPlatform{darwin}}}, ""},
		{"malformed_advertisement", []string{"linux_amd64", "darwin_arm64"}, []tfregistry.ProviderVersion{{Version: "bad", Platforms: both}, {Version: "2", Platforms: both}}, "2.0.0"},
		{"constraint_after_platform_filter", []string{"linux_amd64"}, []tfregistry.ProviderVersion{{Version: "4", Platforms: both}, {Version: "2", Platforms: both}}, "2.0.0"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			x := newConstraintFixture(t, tc.platforms)
			x.catalogs["acme/widget"] = tc.catalog
			f := constraintAST(t, "")
			err := x.apply(t, f, map[string]*tfconfig.ProviderRequirement{"p": constraintRequirement("acme/widget", ">= 2, < 4")})
			if tc.want == "" {
				if err == nil {
					t.Fatal("expected unavailable compatible version")
				}
				if len(f.Body().Blocks()) != 0 {
					t.Fatal("failure added block")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			assertConstraintEntry(t, f, "mirror.example/acme/widget", tc.want, "< 4, >= 2")
			for _, platform := range tc.platforms {
				if !slices.Contains(x.checksumPlatforms, platform) {
					t.Fatalf("missing platform download %s: %v", platform, x.checksumPlatforms)
				}
			}
		})
	}
}

func TestConstraintLockAliasesAndCache(t *testing.T) {
	x := newConstraintFixture(t, nil)
	x.catalogs["acme/widget"] = constraintCatalog("2.1", "2.8", "3.4")
	x.catalogs["other/widget"] = constraintCatalog("5.1")
	req := map[string]*tfconfig.ProviderRequirement{
		"a": constraintRequirement("acme/widget", ">= 2"), "b": constraintRequirement("registry.terraform.io/acme/widget", "< 3"),
		"c": constraintRequirement("mirror.example/acme/widget", "!= 2.8"), "other": constraintRequirement("other/widget", ">= 5"),
		"legacy": constraintRequirement("widget", "nonsense"), "unconstrained": constraintRequirement("acme/unused"),
	}
	f := constraintAST(t, "")
	if err := x.apply(t, f, req); err != nil {
		t.Fatal(err)
	}
	assertConstraintEntry(t, f, "mirror.example/acme/widget", "2.1.0", "!= 2.8, < 3, >= 2")
	assertConstraintEntry(t, f, "mirror.example/other/widget", "5.1.0", ">= 5")
	if len(f.Body().Blocks()) != 2 || len(x.requests) != 2 {
		t.Fatalf("unexpected blocks/requests: %s %v", f.Bytes(), x.requests)
	}
	x.catalogs["acme/widget"] = constraintCatalog("9")
	g := constraintAST(t, "")
	if err := x.apply(t, g, map[string]*tfconfig.ProviderRequirement{"p": constraintRequirement("mirror.example/acme/widget", ">= 3")}); err != nil {
		t.Fatal(err)
	}
	assertConstraintEntry(t, g, "mirror.example/acme/widget", "3.4.0", ">= 3")
	if len(x.requests) != 2 {
		t.Fatalf("catalog fetched twice: %v", x.requests)
	}
	fresh := newConstraintFixture(t, nil)
	fresh.catalogs = x.catalogs
	h := constraintAST(t, "")
	if err := fresh.apply(t, h, map[string]*tfconfig.ProviderRequirement{"p": constraintRequirement("acme/widget", ">= 3")}); err != nil {
		t.Fatal(err)
	}
	assertConstraintEntry(t, h, "mirror.example/acme/widget", "9.0.0", ">= 3")
}

func TestConstraintLockRetryAndEmptyCache(t *testing.T) {
	for _, kind := range []string{"http", "json", "transport", "empty"} {
		t.Run(kind, func(t *testing.T) {
			x := newConstraintFixture(t, nil)
			switch kind {
			case "http":
				x.failures = 1
			case "json":
				x.badJSON = true
			case "transport":
				x.transportError = true
			}
			if kind != "empty" {
				x.catalogs["acme/widget"] = constraintCatalog("2.5")
			}
			req := map[string]*tfconfig.ProviderRequirement{"p": constraintRequirement("acme/widget", ">= 2")}
			f := constraintAST(t, "")
			if err := x.apply(t, f, req); err == nil {
				t.Fatal("expected first request failure")
			}
			x.badJSON = false
			x.transportError = false
			x.catalogs["acme/widget"] = constraintCatalog("2.5")
			err := x.apply(t, f, req)
			if kind == "empty" {
				if err == nil || len(x.requests) != 1 {
					t.Fatalf("successful empty catalog not cached: %v %v", err, x.requests)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			assertConstraintEntry(t, f, "mirror.example/acme/widget", "2.5.0", ">= 2")
			if len(x.requests) != 2 {
				t.Fatalf("failed catalog not retried: %v", x.requests)
			}
		})
	}

	t.Run("checksum", func(t *testing.T) {
		x := newConstraintFixture(t, nil)
		x.checksumFailure = "acme/widget"
		req := map[string]*tfconfig.ProviderRequirement{"p": constraintRequirement("acme/widget", "= 2.5")}
		f := constraintAST(t, "")
		if err := x.apply(t, f, req); err == nil {
			t.Fatal("expected first checksum request failure")
		}
		x.checksumFailure = ""
		if err := x.apply(t, f, req); err != nil {
			t.Fatal(err)
		}
		assertConstraintEntry(t, f, "mirror.example/acme/widget", "2.5.0", "= 2.5")
		if len(x.checksumCalls) != 2 {
			t.Fatalf("failed checksum request not retried: %v", x.checksumCalls)
		}
	})
}

func TestConstraintLockDoesNotExecuteTerraform(t *testing.T) {
	x := newConstraintFixture(t, nil)
	x.catalogs["acme/widget"] = constraintCatalog("2.5")
	dir := t.TempDir()
	marker := filepath.Join(dir, "executed")
	for _, name := range []string{"terraform", "tofu"} {
		script := []byte("#!/bin/sh\nprintf invoked > \"$TFUPDATE_SENTINEL_MARKER\"\nexit 1\n")
		//nolint:gosec // Test sentinels must be executable to detect subprocess calls.
		if err := os.WriteFile(filepath.Join(dir, name), script, 0755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", dir)
	t.Setenv("TFUPDATE_SENTINEL_MARKER", marker)
	f := constraintAST(t, "")
	if err := x.apply(t, f, map[string]*tfconfig.ProviderRequirement{"p": constraintRequirement("acme/widget", ">= 2")}); err != nil {
		t.Fatal(err)
	}
	assertConstraintEntry(t, f, "mirror.example/acme/widget", "2.5.0", ">= 2")
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("Terraform-compatible executable was invoked: %v", err)
	}
}

func TestConstraintLockPreservation(t *testing.T) {
	for _, versionAttr := range []string{"version=\"2.5.0\"", "version=\"1.0.0\"", ""} {
		t.Run(versionAttr, func(t *testing.T) {
			x := newConstraintFixture(t, nil)
			x.catalogs["acme/widget"] = constraintCatalog("2.5")
			input := "# keep comment\nprovider \"mirror.example/acme/widget\" {\n" + versionAttr + "\n constraints=\"old\"\n hashes=[\"h1:preserve\"]\n custom=\"keep\"\n}\nprovider \"other.example/other/unrelated\" {\n version=\"7.0.0\"\n custom=\"other\"\n}\n"
			f := constraintAST(t, input)
			other := string(f.Body().Blocks()[1].BuildTokens(nil).Bytes())
			req := map[string]*tfconfig.ProviderRequirement{"p": constraintRequirement("acme/widget", ">= 2")}
			if err := x.apply(t, f, req); err != nil {
				t.Fatal(err)
			}
			assertConstraintEntry(t, f, "mirror.example/acme/widget", "2.5.0", ">= 2")
			b := f.Body().Blocks()[0]
			if getAttributeValueAsUnquotedString(b.Body().GetAttribute("custom")) != "keep" {
				t.Fatal("custom attribute lost")
			}
			if string(f.Body().Blocks()[1].BuildTokens(nil).Bytes()) != other {
				t.Fatal("unrelated provider changed")
			}
			if !strings.Contains(string(f.Bytes()), "# keep comment") {
				t.Fatal("unrelated comment lost")
			}
			h := string(b.Body().GetAttribute("hashes").BuildTokens(nil).Bytes())
			if strings.Contains(versionAttr, "2.5.0") {
				if len(x.checksumCalls) != 0 || !strings.Contains(h, "h1:preserve") {
					t.Fatalf("unchanged version hashes modified: %s %v", h, x.checksumCalls)
				}
			} else if !strings.Contains(h, "h1:2.5.0") {
				t.Fatalf("new version hashes missing: %s", h)
			}
			before := string(f.Bytes())
			calls := len(x.checksumCalls)
			if err := x.apply(t, f, req); err != nil {
				t.Fatal(err)
			}
			if string(f.Bytes()) != before || len(x.checksumCalls) != calls || len(x.requests) != 1 {
				t.Fatal("repeated update not idempotent or cache missed")
			}
		})
	}
}

func constraintModule(t *testing.T, fs afero.Fs, dir, expression string, lockfile bool) {
	t.Helper()
	if err := fs.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	src := fmt.Sprintf("terraform {\n required_providers {\n widget = { source = \"acme/widget\", version = %q }\n }\n}\n", expression)
	if err := afero.WriteFile(fs, filepath.Join(dir, "main.tf"), []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	if lockfile {
		if err := afero.WriteFile(fs, filepath.Join(dir, ".terraform.lock.hcl"), []byte("# initial\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}
}
func TestConstraintLockFiles(t *testing.T) {
	for _, existing := range []bool{false, true} {
		for _, recursive := range []bool{false, true} {
			t.Run(fmt.Sprintf("existing_%t_recursive_%t", existing, recursive), func(t *testing.T) {
				x := newConstraintFixture(t, nil)
				x.catalogs["acme/widget"] = constraintCatalog("2.5", "3.5")
				fs := afero.NewMemMapFs()
				constraintModule(t, fs, "root", ">= 2, < 3", existing)
				constraintModule(t, fs, "root/child", ">= 3", existing)
				constraintModule(t, fs, "root/ignored", "nonsense", false)
				constraintModule(t, fs, "root/.terraform/modules/downloaded", "nonsense", false)
				if err := afero.WriteFile(fs, "root/call.tf", []byte("module \"remote\" {\n source=\"./missing-child\"\n}\n"), 0644); err != nil {
					t.Fatal(err)
				}
				o, err := NewOption("lock", "", "", nil, recursive, []string{"ignored"}, "", x.config)
				if err != nil {
					t.Fatal(err)
				}
				gc, err := NewGlobalContext(fs, o)
				if err != nil {
					t.Fatal(err)
				}
				gc.updater = x.updater
				if err := UpdateFileOrDir(context.Background(), gc, "root"); err != nil {
					t.Fatal(err)
				}
				data, err := afero.ReadFile(fs, "root/.terraform.lock.hcl")
				if err != nil {
					t.Fatal(err)
				}
				assertConstraintEntry(t, constraintAST(t, string(data)), "mirror.example/acme/widget", "2.5.0", "< 3, >= 2")
				child, err := afero.ReadFile(fs, "root/child/.terraform.lock.hcl")
				if recursive {
					if err != nil {
						t.Fatal(err)
					}
					assertConstraintEntry(t, constraintAST(t, string(child)), "mirror.example/acme/widget", "3.5.0", ">= 3")
				} else if existing {
					if string(child) != "# initial\n" {
						t.Fatal("nonrecursive child changed")
					}
				} else if !os.IsNotExist(err) {
					t.Fatalf("nonrecursive child created: %v", err)
				}
				for _, dir := range []string{"root/ignored", "root/.terraform/modules/downloaded"} {
					if ok, _ := afero.Exists(fs, filepath.Join(dir, ".terraform.lock.hcl")); ok {
						t.Fatalf("ignored lockfile created: %s", dir)
					}
				}
				if len(x.requests) != 1 {
					t.Fatalf("shared catalog calls: %v", x.requests)
				}
				if err := UpdateFileOrDir(context.Background(), gc, "root"); err != nil {
					t.Fatal(err)
				}
				again, _ := afero.ReadFile(fs, "root/.terraform.lock.hcl")
				if !bytes.Equal(data, again) {
					t.Fatal("file update not idempotent")
				}
			})
		}
	}
}

func TestConstraintLockFileErrors(t *testing.T) {
	for _, existing := range []bool{false, true} {
		for _, kind := range []string{"malformed", "no_match", "http", "checksum", "invalid_type"} {
			t.Run(fmt.Sprintf("%s_existing_%t", kind, existing), func(t *testing.T) {
				inputs := []string{kind}
				if kind == "invalid_type" {
					inputs = append(inputs, "invalid_syntax")
				}
				for _, kind := range inputs {
					x := newConstraintFixture(t, nil)
					x.catalogs["acme/widget"] = constraintCatalog("2.5")
					fs := afero.NewMemMapFs()
					expr := ">= 2"
					if kind == "malformed" {
						expr = "nonsense"
					}
					constraintModule(t, fs, "root", expr, existing)
					switch kind {
					case "no_match":
						x.catalogs["acme/widget"] = nil
					case "http":
						x.failures = 1
					case "checksum":
						x.checksumFailure = "widget"
					case "invalid_syntax":
						if err := afero.WriteFile(fs, "root/main.tf", []byte("terraform {\n required_providers {\n widget = "), 0644); err != nil {
							t.Fatal(err)
						}
					case "invalid_type":
						if err := afero.WriteFile(fs, "root/main.tf", []byte("terraform {\n required_providers {\n widget={source=\"acme/widget\",version=[]}\n }\n}\n"), 0644); err != nil {
							t.Fatal(err)
						}
					}
					o := Option{updateType: "lock", tfregistryConfig: x.config}
					gc, err := NewGlobalContext(fs, o)
					if err != nil {
						t.Fatal(err)
					}
					gc.updater = x.updater
					err = UpdateFileOrDir(context.Background(), gc, "root")
					if err == nil {
						t.Fatal("expected contextual module failure")
					}
					if kind == "invalid_type" || kind == "invalid_syntax" {
						assertErrorContext(t, err, "root")
					} else {
						assertProviderErrorContext(t, err, "root", "widget")
					}
					data, e := afero.ReadFile(fs, "root/.terraform.lock.hcl")
					if existing {
						if e != nil || string(data) != "# initial\n" {
							t.Fatalf("failed update changed file: %s %v", data, e)
						}
					} else if !os.IsNotExist(e) {
						t.Fatalf("failed update created file: %s %v", data, e)
					}
				}
			})
		}
	}
}

func TestConstraintLockEmptyAndIgnored(t *testing.T) {
	for _, kind := range []string{"empty", "legacy", "unconstrained", "ignored_lock"} {
		t.Run(kind, func(t *testing.T) {
			x := newConstraintFixture(t, nil)
			x.catalogs["acme/widget"] = constraintCatalog("2.5")
			fs := afero.NewMemMapFs()
			constraintModule(t, fs, "root", ">= 2", false)
			if err := fs.MkdirAll("root/empty", 0755); err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "legacy":
				if err := afero.WriteFile(fs, "root/empty/main.tf", []byte("terraform {\n required_providers {\n widget=\"1.0\"\n }\n}\n"), 0644); err != nil {
					t.Fatal(err)
				}
			case "unconstrained":
				if err := afero.WriteFile(fs, "root/empty/main.tf", []byte("terraform {\n required_providers {\n widget={source=\"acme/widget\"}\n }\n}\n"), 0644); err != nil {
					t.Fatal(err)
				}
			case "ignored_lock":
				constraintModule(t, fs, "root/empty", "nonsense", false)
			}
			ignores := []string{}
			if kind == "ignored_lock" {
				ignores = []string{`empty/\.terraform\.lock\.hcl$`}
			}
			o, err := NewOption("lock", "", "", nil, true, ignores, "", x.config)
			if err != nil {
				t.Fatal(err)
			}
			gc, err := NewGlobalContext(fs, o)
			if err != nil {
				t.Fatal(err)
			}
			gc.updater = x.updater
			if err := UpdateFileOrDir(context.Background(), gc, "root"); err != nil {
				t.Fatal(err)
			}
			data, err := afero.ReadFile(fs, "root/.terraform.lock.hcl")
			if err != nil {
				t.Fatal(err)
			}
			assertConstraintEntry(t, constraintAST(t, string(data)), "mirror.example/acme/widget", "2.5.0", ">= 2")
			if ok, _ := afero.Exists(fs, "root/empty/.terraform.lock.hcl"); ok {
				t.Fatalf("ineligible/ignored lockfile created: %s", kind)
			}
			if len(x.requests) != 1 {
				t.Fatalf("ineligible/ignored module requested catalog: %v", x.requests)
			}
		})
	}
}

func TestConstraintLockRegistryIntegration(t *testing.T) {
	for _, platforms := range [][]string{nil, {"linux_amd64", "darwin_arm64"}} {
		t.Run(fmt.Sprint(platforms), func(t *testing.T) {
			var requests []string
			var zipBuffer bytes.Buffer
			zw := zip.NewWriter(&zipBuffer)
			w, err := zw.Create("provider")
			if err != nil {
				t.Fatal(err)
			}
			if _, err = w.Write([]byte("fixture provider package")); err != nil {
				t.Fatal(err)
			}
			if err = zw.Close(); err != nil {
				t.Fatal(err)
			}
			zipData := zipBuffer.Bytes()
			sum := fmt.Sprintf("%x", sha256.Sum256(zipData))
			config := tfregistry.Config{BaseURL: "https://packages.example/prefix/", HTTPClient: &http.Client{Transport: constraintTransport(func(r *http.Request) (*http.Response, error) {
				requests = append(requests, r.URL.String())
				if r.URL.Host != "packages.example" || !strings.HasPrefix(r.URL.Path, "/prefix/") {
					return nil, fmt.Errorf("incorrect endpoint %s", r.URL)
				}
				var data []byte
				switch {
				case strings.HasSuffix(r.URL.Path, "/versions"):
					data = []byte(`{"versions":[{"version":"2.5.0","platforms":[{"os":"linux","arch":"amd64"},{"os":"darwin","arch":"arm64"}]}]}`)
				case strings.Contains(r.URL.Path, "/download/"):
					data = []byte(fmt.Sprintf(`{"filename":"provider.zip","download_url":"https://packages.example/prefix/package.zip","shasums_url":"https://packages.example/prefix/SHA256SUMS","shasum":%q,"packages":{"linux_amd64":{"hashes":["h1:metadata","zh:metadata"]}}}`, sum))
				case strings.HasSuffix(r.URL.Path, "package.zip"):
					data = zipData
				case strings.HasSuffix(r.URL.Path, "SHA256SUMS"):
					data = []byte(sum + "  provider.zip\n")
				default:
					return nil, fmt.Errorf("unexpected request %s", r.URL)
				}
				return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(data)), Header: make(http.Header)}, nil
			})}}
			u, err := NewLockUpdater(platforms, config)
			if err != nil {
				t.Fatal(err)
			}
			f := constraintAST(t, "")
			mc := &ModuleContext{dir: "module", requiredProviders: map[string]*tfconfig.ProviderRequirement{"p": constraintRequirement("registry.terraform.io/acme/widget", ">= 2")}}
			if err := u.Update(context.Background(), mc, ".terraform.lock.hcl", f); err != nil {
				t.Fatal(err)
			}
			assertConstraintEntry(t, f, "packages.example/acme/widget", "2.5.0", ">= 2")
			previousRequests := len(requests)
			second := constraintAST(t, "")
			mc.requiredProviders["p"] = constraintRequirement("acme/widget", ">= 2")
			if err := u.Update(context.Background(), mc, ".terraform.lock.hcl", second); err != nil {
				t.Fatal(err)
			}
			assertConstraintEntry(t, second, "packages.example/acme/widget", "2.5.0", ">= 2")
			for _, request := range requests[previousRequests:] {
				if strings.HasSuffix(request, "/versions") {
					t.Fatalf("equivalent address missed catalog cache: %v", requests)
				}
			}

			hashes := string(f.Body().Blocks()[0].Body().GetAttribute("hashes").BuildTokens(nil).Bytes())
			if len(platforms) == 0 {
				if !strings.Contains(hashes, "h1:metadata") || previousRequests != 2 {
					t.Fatalf("metadata path: %s %v", hashes, requests)
				}
			} else {
				if !strings.Contains(hashes, "zh:"+sum) || strings.Contains(hashes, "h1:metadata") {
					t.Fatalf("download hashes: %s", hashes)
				}
				for _, suffix := range []string{"download/linux/amd64", "download/darwin/arm64"} {
					found := false
					for _, r := range requests {
						found = found || strings.HasSuffix(r, suffix)
					}
					if !found {
						t.Fatalf("missing platform request: %v", requests)
					}
				}
			}
		})
	}
}

func TestConstraintLockFileTargets(t *testing.T) {
	for _, pathMode := range []string{"file", "multiple_files"} {
		t.Run(pathMode, func(t *testing.T) {
			x := newConstraintFixture(t, nil)
			x.catalogs["acme/widget"] = constraintCatalog("2.1", "2.9", "3.4")
			fs := afero.NewMemMapFs()
			dir := "root"
			constraintModule(t, fs, dir, ">= 2, < 3", true)
			if pathMode == "multiple_files" {
				if err := afero.WriteFile(fs, filepath.Join(dir, "other.tf"), []byte("terraform {\n required_providers {\n alias = {source=\"registry.terraform.io/acme/widget\",version=\"!= 2.9\"}\n }\n}\n"), 0644); err != nil {
					t.Fatal(err)
				}
			}
			gc, err := NewGlobalContext(fs, Option{updateType: "lock", tfregistryConfig: x.config})
			if err != nil {
				t.Fatal(err)
			}
			gc.updater = x.updater
			target := dir
			if pathMode == "file" {
				target = filepath.Join(dir, ".terraform.lock.hcl")
			}
			if err := UpdateFileOrDir(context.Background(), gc, target); err != nil {
				t.Fatal(err)
			}
			data, err := afero.ReadFile(fs, filepath.Join(dir, ".terraform.lock.hcl"))
			if err != nil {
				t.Fatal(err)
			}
			want, record := "2.9.0", "< 3, >= 2"
			if pathMode == "multiple_files" {
				want, record = "2.1.0", "!= 2.9, < 3, >= 2"
			}
			assertConstraintEntry(t, constraintAST(t, string(data)), "mirror.example/acme/widget", want, record)
		})
	}
}

func TestConstraintLockCompatibility(t *testing.T) {
	// Existing non-lock commands must remain independent of registry resolution.
	for _, mode := range []string{"terraform", "opentofu", "provider", "module"} {
		t.Run(mode, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			constraintModule(t, fs, "root", "nonsense", false)
			o, err := NewOption(mode, "acme/widget", "3.0.0", nil, false, nil, "full", tfregistry.Config{})
			if err != nil {
				t.Fatal(err)
			}
			gc, err := NewGlobalContext(fs, o)
			if err != nil {
				t.Fatal(err)
			}
			if err := UpdateFileOrDir(context.Background(), gc, "root"); err != nil {
				t.Fatal(err)
			}
			if ok, _ := afero.Exists(fs, "root/.terraform.lock.hcl"); ok {
				t.Fatal("non-lock command created lockfile")
			}

			x := newConstraintFixture(t, nil)
			x.catalogs["acme/widget"] = constraintCatalog("2.5")
			f := constraintAST(t, "")
			if err := x.apply(t, f, map[string]*tfconfig.ProviderRequirement{"p": constraintRequirement("acme/widget", ">= 2")}); err != nil {
				t.Fatal(err)
			}
			assertConstraintEntry(t, f, "mirror.example/acme/widget", "2.5.0", ">= 2")
		})
	}
}

type constraintObservedStatFS struct {
	afero.Fs
	failed bool
}

func (f *constraintObservedStatFS) Stat(name string) (os.FileInfo, error) {
	if filepath.Base(name) == ".terraform.lock.hcl" {
		f.failed = true
		return nil, os.ErrPermission
	}
	return f.Fs.Stat(name)
}

type constraintInputFailureFS struct {
	afero.Fs
	fault  string
	failed bool
}

func (fs *constraintInputFailureFS) Open(name string) (afero.File, error) {
	return fs.OpenFile(name, os.O_RDONLY, 0)
}

func (fs *constraintInputFailureFS) OpenFile(name string, flag int, mode os.FileMode) (afero.File, error) {
	if fs.fault == "directory" && name == "workspace/module" {
		fs.failed = true
		return nil, os.ErrPermission
	}
	if filepath.Base(name) != ".terraform.lock.hcl" {
		return fs.Fs.OpenFile(name, flag, mode)
	}
	if fs.fault == "open" {
		fs.failed = true
		return nil, os.ErrPermission
	}
	f, err := fs.Fs.OpenFile(name, flag, mode)
	if err != nil {
		return nil, err
	}
	return &constraintInputFailureFile{File: f, fs: fs}, nil
}

type constraintInputFailureFile struct {
	afero.File
	fs *constraintInputFailureFS
}

func (f *constraintInputFailureFile) Read(p []byte) (int, error) {
	if f.fs.fault == "read" {
		f.fs.failed = true
		return 0, os.ErrPermission
	}
	return f.File.Read(p)
}

func (f *constraintInputFailureFile) ReadAt(p []byte, off int64) (int, error) {
	if f.fs.fault == "read" {
		f.fs.failed = true
		return 0, os.ErrPermission
	}
	return f.File.ReadAt(p, off)
}

func (f *constraintInputFailureFile) Close() error {
	err := f.File.Close()
	if err != nil {
		return err
	}
	if f.fs.fault == "close" {
		f.fs.failed = true
		return fmt.Errorf("input close failed")
	}
	return nil
}

func TestConstraintLockInputFailures(t *testing.T) {
	for _, fault := range []string{"open", "read", "close", "parse", "directory", "stat"} {
		t.Run(fault, func(t *testing.T) {
			mem := afero.NewMemMapFs()
			constraintModule(t, mem, "workspace/module", ">= 2", true)
			filename := "workspace/module/.terraform.lock.hcl"
			if fault == "parse" {
				if err := afero.WriteFile(mem, filename, []byte("provider \"acme/widget\" {\n version = (\n}\n"), 0644); err != nil {
					t.Fatal(err)
				}
			}
			if err := mem.Chmod(filename, 0600); err != nil {
				t.Fatal(err)
			}
			before, err := afero.ReadFile(mem, filename)
			if err != nil {
				t.Fatal(err)
			}
			x := newConstraintFixture(t, nil)
			x.catalogs["acme/widget"] = constraintCatalog("2.5")
			gc, err := NewGlobalContext(mem, Option{updateType: "lock", tfregistryConfig: x.config})
			if err != nil {
				t.Fatal(err)
			}
			// Inspection succeeds first; the injected fault affects the subsequent
			// lockfile operation, not the ability to identify its provider.
			mc, err := NewModuleContext("workspace/module", gc)
			if err != nil {
				t.Fatal(err)
			}
			input := &constraintInputFailureFS{Fs: mem, fault: fault}
			metadata := &constraintObservedStatFS{Fs: mem}
			gc.fs = input
			switch fault {
			case "directory":
				err = UpdateDir(context.Background(), mc, "workspace/module")
			case "stat":
				gc.fs = metadata
				err = UpdateFileOrDir(context.Background(), gc, filename)
			default:
				err = UpdateFile(context.Background(), mc, filename)
			}
			after, readErr := afero.ReadFile(mem, filename)
			if input.failed || metadata.failed || fault == "parse" {
				if err == nil {
					t.Fatal("encountered input failure was suppressed")
				}
				assertProviderErrorContext(t, err, "workspace/module", "widget")
				if readErr != nil || !bytes.Equal(before, after) {
					t.Fatalf("input failure changed file: %q, %v", after, readErr)
				}
			} else {
				if err != nil || readErr != nil {
					t.Fatalf("update/read failed without an injected error: %v, %v", err, readErr)
				}
				assertConstraintEntry(t, constraintAST(t, string(after)), "mirror.example/acme/widget", "2.5.0", ">= 2")
			}
			info, statErr := mem.Stat(filename)
			if statErr != nil || info.Mode().Perm() != 0600 {
				t.Fatalf("input handling changed permissions: %v", statErr)
			}
		})
	}
}

// These faults apply to all writable files, independent of staging filenames.
type constraintStorageFS struct {
	afero.Fs
	fault  string
	failed bool
}

func (fs *constraintStorageFS) OpenFile(name string, flag int, mode os.FileMode) (afero.File, error) {
	if flag&(os.O_WRONLY|os.O_RDWR) == 0 {
		return fs.Fs.OpenFile(name, flag, mode)
	}
	if fs.fault == "open" {
		fs.failed = true
		return nil, os.ErrPermission
	}
	f, err := fs.Fs.OpenFile(name, flag, mode)
	if err != nil {
		return nil, err
	}
	return &constraintStorageFile{File: f, fs: fs}, nil
}

func (fs *constraintStorageFS) Create(name string) (afero.File, error) {
	return fs.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
}

func (fs *constraintStorageFS) Rename(oldname, newname string) error {
	if fs.fault == "rename" {
		fs.failed = true
		return os.ErrPermission
	}
	return fs.Fs.Rename(oldname, newname)
}

func (fs *constraintStorageFS) Chmod(name string, mode os.FileMode) error {
	if fs.fault == "chmod" {
		fs.failed = true
		return os.ErrPermission
	}
	return fs.Fs.Chmod(name, mode)
}

type constraintStorageFile struct {
	afero.File
	fs *constraintStorageFS
}

func (f *constraintStorageFile) Write(data []byte) (int, error) {
	if f.fs.fault == "partial" || f.fs.fault == "short" {
		f.fs.failed = true
		n, err := f.File.Write(data[:len(data)/2])
		if err != nil {
			return n, err
		}
		if f.fs.fault == "partial" {
			return n, fmt.Errorf("storage exhausted")
		}
		return n, nil
	}
	return f.File.Write(data)
}

func (f *constraintStorageFile) WriteString(s string) (int, error) {
	return f.Write([]byte(s))
}

func (f *constraintStorageFile) Sync() error {
	if f.fs.fault == "sync" {
		f.fs.failed = true
		return fmt.Errorf("sync failed")
	}
	return f.File.Sync()
}

func (f *constraintStorageFile) Close() error {
	err := f.File.Close()
	if err != nil {
		return err
	}
	if f.fs.fault == "close" {
		f.fs.failed = true
		return fmt.Errorf("close failed")
	}
	return nil
}

func TestConstraintLockPersistence(t *testing.T) {
	for _, target := range []string{"missing", "existing_directory", "existing_file", "multiple_providers"} {
		for _, fault := range []string{"success", "partial", "short", "open", "sync", "close", "rename", "chmod"} {
			t.Run(target+"/"+fault, func(t *testing.T) {
				x := newConstraintFixture(t, nil)
				x.catalogs["acme/widget"] = constraintCatalog("2.5")
				mem := afero.NewMemMapFs()
				existing := target != "missing"
				constraintModule(t, mem, "workspace/module", ">= 2", existing)
				if target == "multiple_providers" {
					x.catalogs["other/tool"] = constraintCatalog("3.1")
					src := "terraform {\n required_providers {\n widget={source=\"acme/widget\", version=\">= 2\"}\n tool={source=\"other/tool\", version=\">= 3\"}\n }\n}\n"
					if err := afero.WriteFile(mem, "workspace/module/main.tf", []byte(src), 0644); err != nil {
						t.Fatal(err)
					}
				}
				filename := "workspace/module/.terraform.lock.hcl"
				before, _ := afero.ReadFile(mem, filename)
				if existing {
					if err := mem.Chmod(filename, 0600); err != nil {
						t.Fatal(err)
					}
				}
				fs := &constraintStorageFS{Fs: mem, fault: fault}
				gc, err := NewGlobalContext(fs, Option{updateType: "lock", tfregistryConfig: x.config})
				if err != nil {
					t.Fatal(err)
				}
				path := "workspace/module"
				if target == "existing_file" {
					path = filename
				}
				err = UpdateFileOrDir(context.Background(), gc, path)
				data, readErr := afero.ReadFile(mem, filename)
				if fs.failed {
					if err == nil {
						t.Fatal("storage failure was suppressed")
					}
					assertProviderErrorContext(t, err, "workspace/module", "widget")
					if target == "multiple_providers" {
						assertProviderErrorContext(t, err, "workspace/module", "tool")
					}
					if existing {
						if readErr != nil || !bytes.Equal(data, before) {
							t.Fatalf("failed update changed lockfile: %q, %v", data, readErr)
						}
					} else if !os.IsNotExist(readErr) {
						t.Fatalf("failed creation left lockfile: %q, %v", data, readErr)
					}
				} else {
					// No particular storage API is required. If the faulted operation
					// wasn't used, demand a complete successful result instead.
					if err != nil || readErr != nil {
						t.Fatalf("update/read failed: %v, %v", err, readErr)
					}
					assertConstraintEntry(t, constraintAST(t, string(data)), "mirror.example/acme/widget", "2.5.0", ">= 2")
					if target == "multiple_providers" {
						assertConstraintEntry(t, constraintAST(t, string(data)), "mirror.example/other/tool", "3.1.0", ">= 3")
					}
				}
				entries, err := afero.ReadDir(mem, "workspace/module")
				if err != nil {
					t.Fatal(err)
				}
				for _, entry := range entries {
					if entry.Name() != "main.tf" && entry.Name() != ".terraform.lock.hcl" {
						t.Errorf("operation left unexpected file %s", entry.Name())
					}
				}
				if existing {
					info, err := mem.Stat(filename)
					if err != nil || info.Mode().Perm() != 0600 {
						t.Errorf("existing lockfile permissions changed: %v", err)
					}
				}
			})
		}
	}
}

func TestConstraintLockExistingStatFailure(t *testing.T) {
	x := newConstraintFixture(t, nil)
	x.catalogs["acme/widget"] = constraintCatalog("2.5")
	mem := afero.NewMemMapFs()
	constraintModule(t, mem, "workspace/module", ">= 2", true)
	filename := "workspace/module/.terraform.lock.hcl"
	before, err := afero.ReadFile(mem, filename)
	if err != nil {
		t.Fatal(err)
	}
	if err := mem.Chmod(filename, 0600); err != nil {
		t.Fatal(err)
	}
	fs := &constraintObservedStatFS{Fs: mem}
	gc, err := NewGlobalContext(fs, Option{updateType: "lock", tfregistryConfig: x.config})
	if err != nil {
		t.Fatal(err)
	}
	err = UpdateFileOrDir(context.Background(), gc, "workspace/module")
	data, readErr := afero.ReadFile(mem, filename)
	if fs.failed {
		if err == nil {
			t.Fatal("encountered metadata failure was suppressed")
		}
		assertProviderErrorContext(t, err, "workspace/module", "widget")
		if readErr != nil || !bytes.Equal(data, before) {
			t.Fatalf("metadata failure changed lockfile: %q, %v", data, readErr)
		}
	} else {
		// Metadata may be obtained through an open file instead of filesystem Stat.
		if err != nil || readErr != nil {
			t.Fatalf("update/read failed without an injected error: %v, %v", err, readErr)
		}
		assertConstraintEntry(t, constraintAST(t, string(data)), "mirror.example/acme/widget", "2.5.0", ">= 2")
	}
	info, statErr := mem.Stat(filename)
	if statErr != nil || info.Mode().Perm() != 0600 {
		t.Fatalf("metadata handling changed permissions: %v", statErr)
	}
	entries, err := afero.ReadDir(mem, "workspace/module")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.Name() != "main.tf" && entry.Name() != ".terraform.lock.hcl" {
			t.Errorf("operation left unexpected file %s", entry.Name())
		}
	}
}

func TestConstraintLockCreationIOErrors(t *testing.T) {
	for _, kind := range []string{"stat", "write"} {
		t.Run(kind, func(t *testing.T) {
			x := newConstraintFixture(t, nil)
			x.catalogs["acme/widget"] = constraintCatalog("2.5")
			mem := afero.NewMemMapFs()
			constraintModule(t, mem, "root", ">= 2", false)
			observed := &constraintObservedStatFS{Fs: mem}
			var fs afero.Fs = observed
			if kind == "write" {
				fs = afero.NewReadOnlyFs(mem)
			}
			gc, err := NewGlobalContext(fs, Option{updateType: "lock", tfregistryConfig: x.config})
			if err != nil {
				t.Fatal(err)
			}
			gc.updater = x.updater
			err = UpdateFileOrDir(context.Background(), gc, "root")
			if err == nil && kind == "stat" && !observed.failed {
				data, readErr := afero.ReadFile(mem, "root/.terraform.lock.hcl")
				if readErr != nil {
					t.Fatal(readErr)
				}
				assertConstraintEntry(t, constraintAST(t, string(data)), "mirror.example/acme/widget", "2.5.0", ">= 2")
				return
			}
			if err == nil {
				t.Fatal("ignored encountered storage error")
			}
			assertProviderErrorContext(t, err, "root", "widget")
			if _, err := mem.Stat("root/.terraform.lock.hcl"); !os.IsNotExist(err) {
				t.Fatalf("failed creation left output: %v", err)
			}
		})
	}
}

func TestConstraintLockExistingFilePreservation(t *testing.T) {
	x := newConstraintFixture(t, nil)
	x.catalogs["acme/widget"] = constraintCatalog("2.5")
	fs := afero.NewMemMapFs()
	constraintModule(t, fs, "root", ">= 2", true)
	input := "# existing file\nprovider \"mirror.example/acme/widget\" {\n version=\"2.5.0\"\n constraints=\"old\"\n hashes=[\"h1:original\"]\n custom=\"keep\"\n}\nprovider \"other.example/team/untouched\" {\n version=\"4.0.0\"\n}\n"
	if err := afero.WriteFile(fs, "root/.terraform.lock.hcl", []byte(input), 0644); err != nil {
		t.Fatal(err)
	}
	gc, err := NewGlobalContext(fs, Option{updateType: "lock", tfregistryConfig: x.config})
	if err != nil {
		t.Fatal(err)
	}
	gc.updater = x.updater
	if err := UpdateFileOrDir(context.Background(), gc, "root"); err != nil {
		t.Fatal(err)
	}
	data, err := afero.ReadFile(fs, "root/.terraform.lock.hcl")
	if err != nil {
		t.Fatal(err)
	}
	f := constraintAST(t, string(data))
	assertConstraintEntry(t, f, "mirror.example/acme/widget", "2.5.0", ">= 2")
	if !strings.Contains(string(data), "# existing file") || !strings.Contains(string(data), "h1:original") || !strings.Contains(string(data), "custom") || f.Body().FirstMatchingBlock("provider", []string{"other.example/team/untouched"}) == nil {
		t.Fatalf("directory update replaced existing contents: %s", data)
	}
	if len(x.checksumCalls) != 0 {
		t.Fatalf("existing same-version entry rehashed: %v", x.checksumCalls)
	}
}

func TestConstraintLockCompactEntries(t *testing.T) {
	inputs := []string{
		`provider "mirror.example/acme/widget" { version = "1.0.0" }`,
		`provider "mirror.example/acme/widget" { version = "2.5.0" }`,
		`provider "mirror.example/acme/widget" {}`,
		`provider "mirror.example/acme/widget" { custom = {inner = "keep"} }`,
		`provider "mirror.example/acme/widget" { /* retain */ custom = "keep" }`,
		"provider \"mirror.example/acme/widget\" {\n custom=\"keep\"\n}",
	}
	for i, input := range inputs {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			x := newConstraintFixture(t, nil)
			x.catalogs["acme/widget"] = constraintCatalog("2.5")
			f := constraintAST(t, input+"\n")
			req := map[string]*tfconfig.ProviderRequirement{"p": constraintRequirement("acme/widget", ">= 2")}
			if err := x.apply(t, f, req); err != nil {
				t.Fatal(err)
			}
			_, d := hclwrite.ParseConfig(f.Bytes(), "out.hcl", hcl.Pos{Line: 1, Column: 1})
			if d.HasErrors() {
				t.Fatalf("invalid generated lockfile: %s\n%s", d, f.Bytes())
			}
			assertConstraintEntry(t, f, "mirror.example/acme/widget", "2.5.0", ">= 2")
			if strings.Contains(input, "keep") && !strings.Contains(string(f.Bytes()), "keep") {
				t.Fatal("custom value lost")
			}
			if strings.Contains(input, "retain") && !strings.Contains(string(f.Bytes()), "retain") {
				t.Fatal("inline comment lost")
			}
			previous := string(f.Bytes())
			if err := x.apply(t, f, req); err != nil {
				t.Fatal(err)
			}
			if string(f.Bytes()) != previous {
				t.Fatalf("compact update not idempotent: %s", f.Bytes())
			}
		})
	}
}

func TestConstraintLockMalformedSyntaxTree(t *testing.T) {
	for _, count := range []int{0, 1, 2} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			x := newConstraintFixture(t, nil)
			x.catalogs["acme/widget"] = constraintCatalog("2.5")
			x.catalogs["acme/other"] = constraintCatalog("3.5")
			req := map[string]*tfconfig.ProviderRequirement{}
			if count > 0 {
				req["widget"] = constraintRequirement("acme/widget", ">= 2")
			}
			if count == 2 {
				req["other"] = constraintRequirement("acme/other", ">= 3")
			}
			f := hclwrite.NewEmptyFile()
			f.Body().AppendUnstructuredTokens(hclwrite.Tokens{{Type: hclsyntax.TokenIdent, Bytes: []byte("broken")}, {Type: hclsyntax.TokenNewline, Bytes: []byte("\n")}})
			before := string(f.BuildTokens(nil).Bytes())
			err := x.apply(t, f, req)
			if err == nil {
				t.Fatal("expected malformed lockfile error")
			}
			assertErrorContext(t, err, "workspace/module")
			if count > 0 {
				assertProviderErrorContext(t, err, "workspace/module", "widget")
			}
			if count == 2 {
				assertProviderErrorContext(t, err, "workspace/module", "other")
			}
			if string(f.BuildTokens(nil).Bytes()) != before {
				t.Fatal("malformed input tree changed")
			}
		})
	}
}

func TestConstraintLockEncodedVersion(t *testing.T) {
	cases := []struct {
		expr     string
		preserve bool
	}{
		{`"\u0032.5.0"`, true},
		{`("2.5.0")`, true},
		{`"${"2.5.0"}"`, true},
		{`"v2.5.0"`, true},
		{`true ? "v2.5" : "1.0"`, true},
		{`"${2}.5"`, true},
		{`2.5`, false},
		{`(2 + 0.5)`, false},
		{`true`, false},
		{`false ? "2.5" : null`, false},
		{`{ major = 2 }`, false},
		{`"not-a-version"`, false},
		{`var.unknown`, false},
		{`null`, false},
		{`["2.5.0"]`, false},
	}
	for _, tc := range cases {
		t.Run(tc.expr, func(t *testing.T) {
			x := newConstraintFixture(t, nil)
			x.catalogs["acme/widget"] = constraintCatalog("2.5")
			f := constraintAST(t, "provider \"mirror.example/acme/widget\" {\n version="+tc.expr+"\n constraints=\"old\"\n hashes=[\"h1:retained\"]\n}\n")
			before := string(f.BuildTokens(nil).Bytes())
			err := x.apply(t, f, map[string]*tfconfig.ProviderRequirement{"p": constraintRequirement("acme/widget", ">= 2")})
			if !tc.preserve {
				if err == nil {
					t.Fatal("expected invalid lockfile error")
				}
				assertProviderErrorContext(t, err, "workspace/module", "widget", "p")
				if string(f.BuildTokens(nil).Bytes()) != before {
					t.Fatal("invalid locked version changed AST")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			assertConstraintEntry(t, f, "mirror.example/acme/widget", "2.5.0", ">= 2")
			hashes := string(f.Body().Blocks()[0].Body().GetAttribute("hashes").BuildTokens(nil).Bytes())
			if len(x.checksumCalls) != 0 || !strings.Contains(hashes, "h1:retained") {
				t.Fatalf("unchanged decoded version rehashed: %s %v", hashes, x.checksumCalls)
			}
		})
	}
}
func TestConstraintLockMalformedVersionExpression(t *testing.T) {
	x := newConstraintFixture(t, nil)
	x.catalogs["acme/widget"] = constraintCatalog("2.5")
	f := hclwrite.NewEmptyFile()
	b := f.Body().AppendNewBlock("provider", []string{"mirror.example/acme/widget"})
	b.Body().SetAttributeRaw("version", hclwrite.Tokens{{Type: hclsyntax.TokenOParen, Bytes: []byte("(")}})
	before := string(f.BuildTokens(nil).Bytes())
	err := x.apply(t, f, map[string]*tfconfig.ProviderRequirement{"p": constraintRequirement("acme/widget", ">= 2")})
	if err == nil {
		t.Fatal("expected malformed-expression error")
	}
	assertProviderErrorContext(t, err, "workspace/module", "widget", "p")
	if string(f.BuildTokens(nil).Bytes()) != before {
		t.Fatal("malformed expression mutated on error")
	}
}

func TestConstraintLockNonStringFileVersion(t *testing.T) {
	for _, expression := range []string{"= 2.5, < 3", ">= 3"} {
		t.Run(expression, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			constraintModule(t, fs, "workspace/module", expression, true)
			filename := "workspace/module/.terraform.lock.hcl"
			before := []byte("provider \"mirror.example/acme/widget\" {\n version = 2.5\n hashes = [\"h1:retained\"]\n}\n")
			if err := afero.WriteFile(fs, filename, before, 0644); err != nil {
				t.Fatal(err)
			}
			x := newConstraintFixture(t, nil)
			x.catalogs["acme/widget"] = constraintCatalog("2.5", "3.2")
			gc, err := NewGlobalContext(fs, Option{updateType: "lock", tfregistryConfig: x.config})
			if err != nil {
				t.Fatal(err)
			}
			err = UpdateFileOrDir(context.Background(), gc, filename)
			if err == nil {
				t.Fatal("numeric locked version was accepted")
			}
			assertProviderErrorContext(t, err, "workspace/module", "widget")
			after, err := afero.ReadFile(fs, filename)
			if err != nil || !bytes.Equal(before, after) {
				t.Fatalf("invalid version changed lockfile: %q, %v", after, err)
			}
		})
	}
}

func TestConstraintLockChecksumTargets(t *testing.T) {
	for _, expression := range []string{"= 2.5, < 3", ">= 2, < 3"} {
		for _, repeated := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/repeated_%t", expression, repeated), func(t *testing.T) {
				platforms := []string{"linux_amd64", "darwin_arm64"}
				if repeated {
					platforms = append(platforms, "linux_amd64", "darwin_arm64")
				}
				x := newConstraintFixture(t, platforms)
				x.catalogs["acme/widget"] = []tfregistry.ProviderVersion{{Version: "2.5", Platforms: []tfregistry.ProviderPlatform{{OS: "linux", Arch: "amd64"}, {OS: "darwin", Arch: "arm64"}}}}
				f := constraintAST(t, "")
				if err := x.apply(t, f, map[string]*tfconfig.ProviderRequirement{"widget": constraintRequirement("acme/widget", expression)}); err != nil {
					t.Fatal(err)
				}
				wantClauses := "< 3, >= 2"
				if strings.HasPrefix(expression, "=") {
					wantClauses = "< 3, = 2.5"
					if len(x.requests) != 0 {
						t.Fatalf("exact pin fetched a catalog: %v", x.requests)
					}
				}
				assertConstraintEntry(t, f, "mirror.example/acme/widget", "2.5.0", wantClauses)
				got := slices.Clone(x.checksumPlatforms)
				slices.Sort(got)
				if !slices.Equal(got, []string{"darwin_arm64", "linux_amd64"}) {
					t.Fatalf("each checksum target must be requested once, got %v", x.checksumPlatforms)
				}
				block := f.Body().FirstMatchingBlock("provider", []string{"mirror.example/acme/widget"})
				if block == nil || block.Body().GetAttribute("hashes") == nil {
					t.Fatal("missing computed hashes")
				}
			})
		}
	}
}

type constraintResponseBody struct {
	io.ReadCloser
	readError  bool
	closeError bool
	closes     int
}

func (b *constraintResponseBody) Read(p []byte) (int, error) {
	if b.readError {
		return 0, io.ErrUnexpectedEOF
	}
	return b.ReadCloser.Read(p)
}

func (b *constraintResponseBody) Close() error {
	b.closes++
	err := b.ReadCloser.Close()
	if b.closeError {
		return fmt.Errorf("injected response close failure: %v", err)
	}
	return err
}

func TestConstraintLockChecksumResponseLifecycle(t *testing.T) {
	cases := []struct {
		endpoint string
		fault    string
	}{
		{"metadata", "503"}, {"metadata", "404"}, {"metadata", "json"}, {"metadata", "read"},
		{"archive", "503"}, {"archive", "read"}, {"checksums", "503"}, {"checksums", "read"},
	}
	for _, expression := range []string{"= 2.5, < 3", ">= 2, < 3"} {
		for _, tc := range cases {
			t.Run(expression+"/"+tc.endpoint+"/"+tc.fault, func(t *testing.T) {
				x := newConstraintFixture(t, []string{"linux_amd64"})
				x.catalogs["acme/widget"] = []tfregistry.ProviderVersion{{Version: "2.5", Platforms: []tfregistry.ProviderPlatform{{OS: "linux", Arch: "amd64"}}}}
				transport := x.config.HTTPClient.Transport
				var bodies []*constraintResponseBody
				inject := true
				injected := 0
				x.config.HTTPClient.Transport = constraintTransport(func(r *http.Request) (*http.Response, error) {
					response, err := transport.RoundTrip(r)
					if err != nil {
						return nil, err
					}
					matches := (tc.endpoint == "metadata" && strings.Contains(r.URL.Path, "/download/")) ||
						(tc.endpoint == "archive" && strings.HasSuffix(r.URL.Path, "/package.zip")) ||
						(tc.endpoint == "checksums" && strings.HasSuffix(r.URL.Path, "/SHA256SUMS"))
					body := &constraintResponseBody{ReadCloser: response.Body}
					if inject && matches {
						injected++
						switch tc.fault {
						case "503":
							response.StatusCode = http.StatusServiceUnavailable
						case "404":
							response.StatusCode = http.StatusNotFound
						case "json":
							if err := response.Body.Close(); err != nil {
								return nil, err
							}
							body.ReadCloser = io.NopCloser(strings.NewReader("{"))
						case "read":
							body.readError = true
						}
					}
					bodies = append(bodies, body)
					response.Body = body
					return response, nil
				})
				f := constraintAST(t, "provider \"mirror.example/acme/widget\" {\n version = \"1.0\"\n hashes = [\"h1:retained\"]\n}\n")
				before := string(f.BuildTokens(nil).Bytes())
				requirements := map[string]*tfconfig.ProviderRequirement{"widget": constraintRequirement("acme/widget", expression)}
				err := x.apply(t, f, requirements)
				if err == nil || injected == 0 {
					t.Fatalf("expected checksum response failure, got %v; injections %d", err, injected)
				}
				assertProviderErrorContext(t, err, "workspace/module", "widget")
				if string(f.BuildTokens(nil).Bytes()) != before {
					t.Fatal("checksum failure changed the input tree")
				}
				for _, body := range bodies {
					if body.closes != 1 {
						t.Errorf("failed attempt closed a response %d times", body.closes)
					}
				}
				inject = false
				bodies = nil
				if err := x.apply(t, f, requirements); err != nil {
					t.Fatalf("retry failed: %v", err)
				}
				wantClauses := "< 3, >= 2"
				if strings.HasPrefix(expression, "=") {
					wantClauses = "< 3, = 2.5"
					if len(x.requests) != 0 {
						t.Fatalf("exact pin fetched a catalog: %v", x.requests)
					}
				}
				assertConstraintEntry(t, f, "mirror.example/acme/widget", "2.5.0", wantClauses)
				if len(bodies) == 0 {
					t.Fatal("checksum failure was cached instead of retried")
				}
				for _, body := range bodies {
					if body.closes != 1 {
						t.Errorf("successful retry closed a response %d times", body.closes)
					}
				}
			})
		}
	}
}

func TestConstraintLockResponseCloseFailures(t *testing.T) {
	for _, endpoint := range []string{"catalog", "metadata", "archive", "checksums"} {
		for _, surface := range []string{"tree", "existing", "missing"} {
			faults := []string{"close", "status-close", "read-close"}
			if endpoint == "catalog" || endpoint == "metadata" {
				faults = append(faults, "json-close")
			}
			for _, fault := range faults {
				t.Run(endpoint+"/"+surface+"/"+fault, func(t *testing.T) {
					x := newConstraintFixture(t, []string{"linux_amd64"})
					x.catalogs["acme/widget"] = []tfregistry.ProviderVersion{{Version: "2.5", Platforms: []tfregistry.ProviderPlatform{{OS: "linux", Arch: "amd64"}}}}
					transport := x.config.HTTPClient.Transport
					inject := true
					injected := 0
					var bodies []*constraintResponseBody
					x.config.HTTPClient.Transport = constraintTransport(func(r *http.Request) (*http.Response, error) {
						response, err := transport.RoundTrip(r)
						if err != nil {
							return nil, err
						}
						matches := (endpoint == "catalog" && strings.HasSuffix(r.URL.Path, "/versions")) ||
							(endpoint == "metadata" && strings.Contains(r.URL.Path, "/download/")) ||
							(endpoint == "archive" && strings.HasSuffix(r.URL.Path, "/package.zip")) ||
							(endpoint == "checksums" && strings.HasSuffix(r.URL.Path, "/SHA256SUMS"))
						body := &constraintResponseBody{ReadCloser: response.Body}
						if inject && matches {
							injected++
							body.closeError = true
							if fault == "json-close" {
								if err := body.ReadCloser.Close(); err != nil {
									return nil, err
								}
								body.ReadCloser = io.NopCloser(strings.NewReader("{"))
							}
							if fault == "status-close" {
								response.StatusCode = http.StatusServiceUnavailable
								response.Status = "503 Service Unavailable"
							}
							if fault == "read-close" {
								body.readError = true
							}
						}
						bodies = append(bodies, body)
						response.Body = body
						return response, nil
					})
					// Catalog errors need ranges; checksum stages also exercise catalog-free pins.
					expression := "= 2.5"
					if endpoint == "catalog" {
						expression = ">= 2"
					}
					input := "provider \"mirror.example/acme/widget\" {\n version=\"1.0\"\n hashes=[\"h1:retained\"]\n}\n"
					f := constraintAST(t, input)
					before := string(f.BuildTokens(nil).Bytes())
					fs := afero.NewMemMapFs()
					constraintModule(t, fs, "workspace/module", expression, surface == "existing")
					filename := "workspace/module/.terraform.lock.hcl"
					if surface == "existing" {
						if err := afero.WriteFile(fs, filename, []byte(input), 0600); err != nil {
							t.Fatal(err)
						}
						if err := fs.Chmod(filename, 0600); err != nil {
							t.Fatal(err)
						}
					}
					gc, err := NewGlobalContext(fs, Option{updateType: "lock", tfregistryConfig: x.config, platforms: []string{"linux_amd64"}})
					if err != nil {
						t.Fatal(err)
					}
					gc.updater = x.updater
					run := func() error {
						if surface == "tree" {
							return x.apply(t, f, map[string]*tfconfig.ProviderRequirement{"widget": constraintRequirement("acme/widget", expression)})
						}
						return UpdateFileOrDir(context.Background(), gc, "workspace/module")
					}
					err = run()
					if err == nil || injected == 0 {
						t.Fatalf("expected response lifecycle error, got %v; injections %d", err, injected)
					}
					assertProviderErrorContext(t, err, "workspace/module", "widget")
					if !strings.Contains(err.Error(), "injected response close failure") {
						t.Fatalf("close failure was discarded: %v", err)
					}
					// Both causes must remain observable; do not prescribe errors.Join
					// or a particular wrapping/message format.
					message := strings.ToLower(err.Error())
					switch fault {
					case "status-close":
						if !strings.Contains(message, "503") && !strings.Contains(message, "service unavailable") {
							t.Fatalf("HTTP status failure was discarded: %v", err)
						}
					case "read-close":
						if !strings.Contains(message, strings.ToLower(io.ErrUnexpectedEOF.Error())) {
							t.Fatalf("body read failure was discarded: %v", err)
						}
					case "json-close":
						// An unfinished JSON object must retain its truncation cause.
						if !strings.Contains(message, "eof") && !strings.Contains(message, "end of") &&
							!strings.Contains(message, "unterminated") && !strings.Contains(message, "incomplete") {
							t.Fatalf("JSON truncation failure was discarded: %v", err)
						}
					}
					if string(f.BuildTokens(nil).Bytes()) != before {
						t.Fatal("response failure changed tree")
					}
					switch surface {
					case "existing":
						data, err := afero.ReadFile(fs, filename)
						if err != nil || string(data) != input {
							t.Fatalf("response failure changed bytes: %q, %v", data, err)
						}
						info, err := fs.Stat(filename)
						if err != nil || info.Mode().Perm() != 0600 {
							t.Fatalf("response failure changed permissions: %v", err)
						}
					case "missing":
						if _, err := fs.Stat(filename); !os.IsNotExist(err) {
							t.Fatalf("failed request created lockfile: %v", err)
						}
					}
					for _, body := range bodies {
						if body.closes != 1 {
							t.Errorf("response closed %d times", body.closes)
						}
					}
					inject = false
					bodies = nil
					previousCatalogs := len(x.requests)
					if err := run(); err != nil {
						t.Fatalf("retry failed: %v", err)
					}
					if endpoint == "catalog" && len(x.requests) <= previousCatalogs {
						t.Fatal("failed catalog was cached")
					}
					if surface != "tree" {
						data, err := afero.ReadFile(fs, filename)
						if err != nil {
							t.Fatal(err)
						}
						f = constraintAST(t, string(data))
					}
					assertConstraintEntry(t, f, "mirror.example/acme/widget", "2.5.0", expression)
					if len(bodies) == 0 {
						t.Fatal("failed checksum request was cached")
					}
					for _, body := range bodies {
						if body.closes != 1 {
							t.Errorf("retry response closed %d times", body.closes)
						}
					}
				})
			}
		}
	}
}

// Observe recursive source failures through either supported read API.
type constraintChildReadFailureFS struct {
	afero.Fs
	filename    string
	encountered bool
}

func (fs *constraintChildReadFailureFS) wrap(name string, f afero.File, err error) (afero.File, error) {
	if err != nil {
		return nil, err
	}
	if filepath.Clean(name) == filepath.Clean(fs.filename) {
		return &constraintChildReadFailureFile{File: f, encountered: &fs.encountered}, nil
	}
	return f, nil
}

func (fs *constraintChildReadFailureFS) Open(name string) (afero.File, error) {
	f, err := fs.Fs.Open(name)
	return fs.wrap(name, f, err)
}

func (fs *constraintChildReadFailureFS) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	f, err := fs.Fs.OpenFile(name, flag, perm)
	return fs.wrap(name, f, err)
}

type constraintChildReadFailureFile struct {
	afero.File
	encountered *bool
}

func (f *constraintChildReadFailureFile) Read(_ []byte) (int, error) {
	*f.encountered = true
	return 0, os.ErrPermission
}

func (f *constraintChildReadFailureFile) ReadAt(_ []byte, _ int64) (int, error) {
	*f.encountered = true
	return 0, os.ErrPermission
}

func TestConstraintLockRecursiveChildErrorContext(t *testing.T) {
	for _, rootProvider := range []bool{false, true} {
		t.Run(fmt.Sprint(rootProvider), func(t *testing.T) {
			mem := afero.NewMemMapFs()
			constraintModule(t, mem, "root/child", ">= 2", true)
			if rootProvider {
				input := "terraform {\n required_providers {\n parent = {source=\"other/parent\",version=\">= 1\"}\n }\n}\n"
				if err := afero.WriteFile(mem, "root/main.tf", []byte(input), 0644); err != nil {
					t.Fatal(err)
				}
			}
			filename := "root/child/.terraform.lock.hcl"
			if err := mem.Chmod(filename, 0600); err != nil {
				t.Fatal(err)
			}
			before, err := afero.ReadFile(mem, filename)
			if err != nil {
				t.Fatal(err)
			}
			x := newConstraintFixture(t, nil)
			x.catalogs["acme/widget"] = constraintCatalog("2.5")
			x.catalogs["other/parent"] = constraintCatalog("1.5")
			fs := &constraintChildReadFailureFS{Fs: mem, filename: filename}
			gc, err := NewGlobalContext(fs, Option{updateType: "lock", recursive: true, tfregistryConfig: x.config})
			if err != nil {
				t.Fatal(err)
			}
			err = UpdateFileOrDir(context.Background(), gc, "root")
			after, readErr := afero.ReadFile(mem, filename)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if !fs.encountered && err == nil {
				// A strategy that does not use the failing operation must still
				// complete the requested child update.
				assertConstraintEntry(t, constraintAST(t, string(after)), "mirror.example/acme/widget", "2.5.0", ">= 2")
			} else {
				if err == nil {
					t.Fatal("expected child lockfile read failure")
				}
				assertProviderErrorContext(t, err, "root/child", "widget")
				if !bytes.Equal(before, after) {
					t.Fatalf("child read failure changed bytes: %q", after)
				}
			}
			info, err := mem.Stat(filename)
			if err != nil || info.Mode().Perm() != 0600 {
				t.Fatalf("child read failure changed permissions: %v", err)
			}
		})
	}
}

func TestConstraintLockUnconstrainedMalformedSource(t *testing.T) {
	for _, existing := range []bool{false, true} {
		t.Run(fmt.Sprint(existing), func(t *testing.T) {
			mem := afero.NewMemMapFs()
			constraintModule(t, mem, "root", ">= 2", existing)
			input := "terraform {\n required_providers {\n widget={source=\"acme/widget\",version=\">= 2\"}\n broken={source=\"not a provider address\"}\n }\n}\n"
			if err := afero.WriteFile(mem, "root/main.tf", []byte(input), 0644); err != nil {
				t.Fatal(err)
			}
			x := newConstraintFixture(t, nil)
			x.catalogs["acme/widget"] = constraintCatalog("2.5")
			gc, err := NewGlobalContext(mem, Option{updateType: "lock", tfregistryConfig: x.config})
			if err != nil {
				t.Fatal(err)
			}
			if err := UpdateFileOrDir(context.Background(), gc, "root"); err != nil {
				t.Fatal(err)
			}
			data, err := afero.ReadFile(mem, "root/.terraform.lock.hcl")
			if err != nil {
				t.Fatal(err)
			}
			f := constraintAST(t, string(data))
			assertConstraintEntry(t, f, "mirror.example/acme/widget", "2.5.0", ">= 2")
			if len(f.Body().Blocks()) != 1 {
				t.Fatalf("ignored declaration produced lock entry: %s", data)
			}
			if len(x.requests) != 1 || !strings.Contains(x.requests[0], "/acme/widget/versions") {
				t.Fatalf("unexpected catalogs: %v", x.requests)
			}
			for _, request := range x.checksumCalls {
				if request != "acme/widget@2.5.0" {
					t.Fatalf("ignored provider checksum request: %s", request)
				}
			}
		})
	}
}

func TestConstraintLockAllProviderPlanningErrors(t *testing.T) {
	for _, target := range []string{"memory", "missing", "directory", "file", "source"} {
		for _, fault := range []string{"constraint", "catalog", "checksum", "locked_version", "locked_expression", "http", "inspection"} {
			if (target == "missing" && strings.HasPrefix(fault, "locked_")) || (target == "memory" && fault == "inspection") || (target == "source" && fault != "inspection") {
				continue
			}
			t.Run(target+"/"+fault, func(t *testing.T) {
				x := newConstraintFixture(t, nil)
				x.catalogs["acme/alpha"] = constraintCatalog("2.5")
				x.catalogs["acme/bravo"] = constraintCatalog("3.5")
				bravo := ">= 3"
				if fault == "constraint" {
					bravo = "not a constraint"
				}
				if fault == "catalog" {
					x.catalogs["acme/bravo"] = nil
				}
				if fault == "checksum" {
					x.checksumFailure = "bravo"
				}
				if fault == "http" {
					x.failures = 1
				}
				input := "provider \"mirror.example/acme/alpha\" {\n version=\"2.5.0\"\n hashes=[\"h1:keep\"]\n}\n"
				if fault == "locked_version" {
					input += "provider \"mirror.example/acme/bravo\" { version=\"not-a-version\" }\n"
				}
				if fault == "locked_expression" {
					input += "provider \"mirror.example/acme/bravo\" { version=3.5 }\n"
				}
				dir := "workspace/module"
				if target == "memory" {
					f := constraintAST(t, input)
					before := string(f.BuildTokens(nil).Bytes())
					mc := &ModuleContext{dir: dir, requiredProviders: map[string]*tfconfig.ProviderRequirement{
						"alpha": constraintRequirement("acme/alpha", "= 2.5"),
						"bravo": constraintRequirement("acme/bravo", bravo),
					}}
					err := x.updater.Update(context.Background(), mc, ".terraform.lock.hcl", f)
					assertProviderErrorContext(t, err, dir, "alpha")
					assertProviderErrorContext(t, err, dir, "bravo")
					if string(f.BuildTokens(nil).Bytes()) != before {
						t.Fatal("planning failure changed supplied tree")
					}
					return
				}
				mem := afero.NewMemMapFs()
				if err := mem.MkdirAll(dir, 0755); err != nil {
					t.Fatal(err)
				}
				config := fmt.Sprintf("terraform {\n required_providers {\n alpha={source=\"acme/alpha\",version=\"= 2.5\"}\n bravo={source=\"acme/bravo\",version=%q}\n }\n}\n", bravo)
				if err := afero.WriteFile(mem, filepath.Join(dir, "main.tf"), []byte(config), 0644); err != nil {
					t.Fatal(err)
				}
				if fault == "inspection" {
					if err := afero.WriteFile(mem, filepath.Join(dir, "broken.tf"), []byte("terraform {"), 0644); err != nil {
						t.Fatal(err)
					}
				}
				filename := filepath.Join(dir, ".terraform.lock.hcl")
				if target != "missing" {
					if err := afero.WriteFile(mem, filename, []byte(input), 0600); err != nil {
						t.Fatal(err)
					}
				}
				gc, err := NewGlobalContext(mem, Option{updateType: "lock", tfregistryConfig: x.config})
				if err != nil {
					t.Fatal(err)
				}
				path := dir
				if target == "file" {
					path = filename
				}
				if target == "source" {
					path = filepath.Join(dir, "broken.tf")
				}
				err = UpdateFileOrDir(context.Background(), gc, path)
				assertProviderErrorContext(t, err, dir, "alpha")
				assertProviderErrorContext(t, err, dir, "bravo")
				data, readErr := afero.ReadFile(mem, filename)
				if target == "missing" {
					if !os.IsNotExist(readErr) {
						t.Fatalf("failed creation left lockfile: %s, %v", data, readErr)
					}
				} else {
					if readErr != nil || string(data) != input {
						t.Fatalf("planning failure changed bytes: %q, %v", data, readErr)
					}
					info, err := mem.Stat(filename)
					if err != nil || info.Mode().Perm() != 0600 {
						t.Fatalf("planning failure changed permissions: %v", err)
					}
				}
			})
		}
	}
}

type constraintLateSourceCloseFS struct {
	afero.Fs
	armed  bool
	failed bool
}

func (fs *constraintLateSourceCloseFS) wrap(name string, f afero.File, err error) (afero.File, error) {
	if err != nil {
		return nil, err
	}
	if filepath.Base(name) == "main.tf" {
		return &constraintLateSourceCloseFile{File: f, fs: fs}, nil
	}
	return f, nil
}

func (fs *constraintLateSourceCloseFS) Open(name string) (afero.File, error) {
	f, err := fs.Fs.Open(name)
	return fs.wrap(name, f, err)
}

func (fs *constraintLateSourceCloseFS) OpenFile(name string, flag int, mode os.FileMode) (afero.File, error) {
	f, err := fs.Fs.OpenFile(name, flag, mode)
	return fs.wrap(name, f, err)
}

type constraintLateSourceCloseFile struct {
	afero.File
	fs *constraintLateSourceCloseFS
}

func (f *constraintLateSourceCloseFile) Close() error {
	err := f.File.Close()
	if f.fs.armed {
		f.fs.failed = true
		return fmt.Errorf("source close failure: %v", err)
	}
	return err
}

func TestConstraintLockSourceReadAtomicity(t *testing.T) {
	mem := afero.NewMemMapFs()
	constraintModule(t, mem, "root", ">= 2", true)
	filename := "root/.terraform.lock.hcl"
	if err := mem.Chmod(filename, 0600); err != nil {
		t.Fatal(err)
	}
	before, err := afero.ReadFile(mem, filename)
	if err != nil {
		t.Fatal(err)
	}
	fs := &constraintLateSourceCloseFS{Fs: mem}
	x := newConstraintFixture(t, nil)
	x.catalogs["acme/widget"] = constraintCatalog("2.5")
	transport := x.config.HTTPClient.Transport
	x.config.HTTPClient.Transport = constraintTransport(func(req *http.Request) (*http.Response, error) {
		fs.armed = true
		return transport.RoundTrip(req)
	})
	gc, err := NewGlobalContext(fs, Option{updateType: "lock", tfregistryConfig: x.config})
	if err != nil {
		t.Fatal(err)
	}
	err = UpdateFileOrDir(context.Background(), gc, "root")
	after, readErr := afero.ReadFile(mem, filename)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if fs.failed {
		assertProviderErrorContext(t, err, "root", "widget")
		if !bytes.Equal(before, after) {
			t.Fatal("late source error left committed lockfile")
		}
	} else {
		if err != nil {
			t.Fatal(err)
		}
		assertConstraintEntry(t, constraintAST(t, string(after)), "mirror.example/acme/widget", "2.5.0", ">= 2")
	}
	info, err := mem.Stat(filename)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatalf("lockfile permissions changed: %v", err)
	}
}

// Exercise each documented configuration default without leaving the process.
// These subtests are sequential because net/http's default transport is global.
func TestConstraintLockRegistryDefaults(t *testing.T) {
	original := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = original })
	for _, mode := range []string{"empty", "endpoint_only", "client_only", "endpoint_and_client"} {
		t.Run(mode, func(t *testing.T) {
			x := newConstraintFixture(t, nil)
			x.catalogs["acme/widget"] = constraintCatalog("3.0", "2.3", "2.5")
			x.failures = 1
			baseURL, host, prefix := "", "registry.terraform.io", "/"
			if strings.HasPrefix(mode, "endpoint") {
				baseURL, host, prefix = "https://custom.example/api/", "custom.example", "/api/"
			}
			offline := constraintTransport(func(req *http.Request) (*http.Response, error) {
				if req.URL.Scheme != "https" || req.URL.Host != host || !strings.HasPrefix(req.URL.Path, prefix+"v1/providers/") {
					return nil, fmt.Errorf("unexpected registry endpoint: %s", req.URL)
				}
				forwarded := req.Clone(req.Context())
				forwarded.URL.Host = "mirror.example"
				forwarded.URL.Path = "/prefix/" + strings.TrimPrefix(req.URL.Path, prefix)
				return x.config.HTTPClient.Transport.RoundTrip(forwarded)
			})
			config := tfregistry.Config{BaseURL: baseURL}
			http.DefaultTransport = offline
			if strings.Contains(mode, "client") {
				config.HTTPClient = &http.Client{Transport: offline}
				http.DefaultTransport = constraintTransport(func(req *http.Request) (*http.Response, error) {
					return nil, fmt.Errorf("configured client was bypassed: %s", req.URL)
				})
			}
			u, err := NewLockUpdater(nil, config)
			if err != nil {
				t.Fatal(err)
			}
			mc := &ModuleContext{dir: "workspace/module", requiredProviders: map[string]*tfconfig.ProviderRequirement{
				"widget": constraintRequirement("acme/widget", "~> 2.0", ">= 2.3"),
			}}
			f := constraintAST(t, "# keep this comment\n")
			before := string(f.Bytes())
			err = u.Update(context.Background(), mc, ".terraform.lock.hcl", f)
			if err == nil {
				t.Fatal("default or partial configuration skipped a failed range lookup")
			}
			assertProviderErrorContext(t, err, "workspace/module", "widget")
			if string(f.Bytes()) != before {
				t.Fatal("failed lookup changed the supplied tree")
			}
			if err := u.Update(context.Background(), mc, ".terraform.lock.hcl", f); err != nil {
				t.Fatal(err)
			}
			assertConstraintEntry(t, f, host+"/acme/widget", "2.5.0", ">= 2.3, ~> 2.0")
			block := f.Body().FirstMatchingBlock("provider", []string{host + "/acme/widget"})
			if block == nil || block.Body().GetAttribute("hashes") == nil || len(x.checksumCalls) == 0 {
				t.Fatal("resolved provider lacks checksum work")
			}
			if len(x.requests) != 2 {
				t.Fatalf("failed catalog was not retried: %v", x.requests)
			}
			stable := string(f.Bytes())
			if err := u.Update(context.Background(), mc, ".terraform.lock.hcl", f); err != nil {
				t.Fatal(err)
			}
			if len(x.requests) != 2 || string(f.Bytes()) != stable {
				t.Fatal("successful catalog reuse or idempotence failed")
			}
		})
	}
}

func TestConstraintLockConstrainedMalformedSource(t *testing.T) {
	for _, source := range []string{"bad host/acme/widget", "registry.example/acme/widget/extra"} {
		t.Run(source, func(t *testing.T) {
			x := newConstraintFixture(t, nil)
			x.catalogs["acme/widget"] = constraintCatalog("2.5")
			f := constraintAST(t, "")
			requirements := map[string]*tfconfig.ProviderRequirement{
				"broken": constraintRequirement(source, ">= 2"),
				"widget": constraintRequirement("acme/widget", ">= 2"),
			}
			if err := x.apply(t, f, requirements); err != nil {
				t.Fatal(err)
			}
			assertConstraintEntry(t, f, "mirror.example/acme/widget", "2.5.0", ">= 2")
			if len(f.Body().Blocks()) != 1 || len(x.requests) != 1 || !strings.Contains(x.requests[0], "/acme/widget/versions") {
				t.Fatalf("malformed source was resolved: %s, %v", f.Bytes(), x.requests)
			}
			for _, request := range x.checksumCalls {
				if request != "acme/widget@2.5.0" {
					t.Fatalf("malformed source checksum request: %s", request)
				}
			}
		})
	}
}

func TestUpdateLockWithOpenTofuRegistry(t *testing.T) {
	platforms := []string{"darwin_arm64", "darwin_amd64", "linux_amd64"}
	pvs := []*lock.ProviderVersion{
		lock.NewMockProviderVersion(
			"hashicorp/null",
			"3.2.1",
			[]string{"darwin_arm64", "darwin_amd64", "linux_amd64"},
			map[string]string{

				"terraform-provider-null_3.2.1_linux_amd64.zip":  "h1:uQv2oPjJv+ue8bPrVp+So2hHd90UTssnCNajTW554Cw=",
				"terraform-provider-null_3.2.1_darwin_amd64.zip": "h1:BdIx478D4wpFgFeaS+5Bdve1HIlt3Yhsr5hAbUs2rRg=",
				"terraform-provider-null_3.2.1_darwin_arm64.zip": "h1:+JAon/4CyriC/c7c77NjJalKrKx6gwwQ7L7rVABWMtA=",
			},
			map[string]string{
				"terraform-provider-null_3.2.1_linux_amd64.zip":   "zh:40335019c11e5bdb3780301da5197cbc756b4b5ac3d699c52583f1d34e963176",
				"terraform-provider-null_3.2.1_linux_arm.zip":     "zh:42356e687656fc8ec1f230f786f830f344e64419552ec483e2bc79bd4b7cf1e8",
				"terraform-provider-null_3.2.1_darwin_arm64.zip":  "zh:5ce03460813954cbebc9f9ad5befbe364d9dc67acb08869f67c1aa634fbf6d6c",
				"terraform-provider-null_3.2.1_windows_386.zip":   "zh:658ea3e3e7ecc964bdbd08ecde63f3d79f298bab9922b29a6526ba941a4d403a",
				"terraform-provider-null_3.2.1_windows_arm64.zip": "zh:68c06703bc57f9c882bfedda6f3047775f0d367093d00efb040800c798b8a613",
				"terraform-provider-null_3.2.1_windows_amd64.zip": "zh:80fd03335f793dc54302dd53da98c91fd94f182bcacf13457bed1a99ecffbc1a",
				"terraform-provider-null_3.2.1_linux_arm64.zip":   "zh:91a76f371815a130735c8fcb6196804d878aebcc67b4c3b73571d2063336ffd8",
				"terraform-provider-null_3.2.1_windows_arm.zip":   "zh:c146fc0291b7f6284fe4d54ce6aaece6957e9acc93fc572dd505dfd8bcad3e6c",
				"terraform-provider-null_3.2.1_linux_386.zip":     "zh:c38c9a295cfae9fb6372523c34b9466cd439d5e2c909b56a788960d387c24320",
				"terraform-provider-null_3.2.1_darwin_amd64.zip":  "zh:e25265d4e87821d18dc9653a0ce01978a1ae5d363bc01dd273454db1aa0309c7",
			},
		),
	}

	cases := []struct {
		desc             string
		tfregistryConfig tfregistry.Config
		src              string
		lockfile         string
		want             string
		ok               bool
	}{
		{
			desc: "simple",
			tfregistryConfig: tfregistry.Config{
				BaseURL: "https://registry.opentofu.org/",
			},
			src: `
terraform {
  required_version = "1.9.0"

  required_providers {
    null = {
      source  = "hashicorp/null"
      version = "3.2.1"
    }
  }
}
`,
			lockfile: `
# This file is maintained automatically by "tofu init".
# Manual edits may be lost in future updates.

provider "registry.opentofu.org/hashicorp/null" {
  version     = "3.1.1"
  constraints = "3.1.1"
  hashes = [
    "h1:Difmh8E1BLxb+7+waC9B76cMdsrcjaHlSY4r0fF2pwo=",
    "h1:QitqPlS79NODAC5hplISDtI7b84ETNx3H9nrcJfMgF8=",
    "h1:r1SUdAxGh9oAwW1XRSsbfIxGQo29cRvuLxxICN3KjWI=",
    "zh:35e3cd3af991a2d6f5c3818d44c4d2c188238782849f79c8ba30caa2f73985a7",
    "zh:4b1451dcd6a641a7e618c2bb586bffb8438d9142c357afb645266b94ec1a8450",
    "zh:4cc1c7774562bed41d6bac9cfe80d81a2b68c6a13594058ae21d66ce0854db6e",
    "zh:63535ceefb44f94482c8b6cf70603d64f966c17772e8e2402dd879b54523396a",
    "zh:70de13ff0371960286870f14e7d89d0dd886c2418aaaac8045ed82c8787ceb14",
    "zh:79f158d4ac8c25915312a2bc1133c70902974f369a7cedd652439184487ad2c3",
    "zh:7f38e45a859fc82e9122a8d01d568c664f723910cc6714421640134a9f974681",
    "zh:b5b7fc2833c5c7c2f1bb7cda9dca8c2d34ea2ed8d592cd870d20ae468d1c5650",
    "zh:f4183aea2c36dfb015990af272c55606345ddbc586136628b11687f3e99c95a4",
  ]
}
`,
			want: `
# This file is maintained automatically by "tofu init".
# Manual edits may be lost in future updates.

provider "registry.opentofu.org/hashicorp/null" {
  version     = "3.2.1"
  constraints = "3.2.1"
  hashes = [
    "h1:+JAon/4CyriC/c7c77NjJalKrKx6gwwQ7L7rVABWMtA=",
    "h1:BdIx478D4wpFgFeaS+5Bdve1HIlt3Yhsr5hAbUs2rRg=",
    "h1:uQv2oPjJv+ue8bPrVp+So2hHd90UTssnCNajTW554Cw=",
    "zh:40335019c11e5bdb3780301da5197cbc756b4b5ac3d699c52583f1d34e963176",
    "zh:42356e687656fc8ec1f230f786f830f344e64419552ec483e2bc79bd4b7cf1e8",
    "zh:5ce03460813954cbebc9f9ad5befbe364d9dc67acb08869f67c1aa634fbf6d6c",
    "zh:658ea3e3e7ecc964bdbd08ecde63f3d79f298bab9922b29a6526ba941a4d403a",
    "zh:68c06703bc57f9c882bfedda6f3047775f0d367093d00efb040800c798b8a613",
    "zh:80fd03335f793dc54302dd53da98c91fd94f182bcacf13457bed1a99ecffbc1a",
    "zh:91a76f371815a130735c8fcb6196804d878aebcc67b4c3b73571d2063336ffd8",
    "zh:c146fc0291b7f6284fe4d54ce6aaece6957e9acc93fc572dd505dfd8bcad3e6c",
    "zh:c38c9a295cfae9fb6372523c34b9466cd439d5e2c909b56a788960d387c24320",
    "zh:e25265d4e87821d18dc9653a0ce01978a1ae5d363bc01dd273454db1aa0309c7",
  ]
}
`,
			ok: true,
		},
		{
			desc: "create from empty",
			tfregistryConfig: tfregistry.Config{
				BaseURL: "https://registry.opentofu.org/",
			},
			src: `
terraform {
  required_version = "1.9.0"

  required_providers {
    null = {
      source  = "hashicorp/null"
      version = "3.2.1"
    }
  }
}
`,
			lockfile: ``,
			want: `
provider "registry.opentofu.org/hashicorp/null" {
  version     = "3.2.1"
  constraints = "3.2.1"
  hashes = [
    "h1:+JAon/4CyriC/c7c77NjJalKrKx6gwwQ7L7rVABWMtA=",
    "h1:BdIx478D4wpFgFeaS+5Bdve1HIlt3Yhsr5hAbUs2rRg=",
    "h1:uQv2oPjJv+ue8bPrVp+So2hHd90UTssnCNajTW554Cw=",
    "zh:40335019c11e5bdb3780301da5197cbc756b4b5ac3d699c52583f1d34e963176",
    "zh:42356e687656fc8ec1f230f786f830f344e64419552ec483e2bc79bd4b7cf1e8",
    "zh:5ce03460813954cbebc9f9ad5befbe364d9dc67acb08869f67c1aa634fbf6d6c",
    "zh:658ea3e3e7ecc964bdbd08ecde63f3d79f298bab9922b29a6526ba941a4d403a",
    "zh:68c06703bc57f9c882bfedda6f3047775f0d367093d00efb040800c798b8a613",
    "zh:80fd03335f793dc54302dd53da98c91fd94f182bcacf13457bed1a99ecffbc1a",
    "zh:91a76f371815a130735c8fcb6196804d878aebcc67b4c3b73571d2063336ffd8",
    "zh:c146fc0291b7f6284fe4d54ce6aaece6957e9acc93fc572dd505dfd8bcad3e6c",
    "zh:c38c9a295cfae9fb6372523c34b9466cd439d5e2c909b56a788960d387c24320",
    "zh:e25265d4e87821d18dc9653a0ce01978a1ae5d363bc01dd273454db1aa0309c7",
  ]
}
`,
			ok: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			dirname := "test"
			err := fs.MkdirAll(dirname, os.ModePerm)
			if err != nil {
				t.Fatalf("failed to create dir: %s", err)
			}

			err = afero.WriteFile(fs, filepath.Join(dirname, "main.tf"), []byte(tc.src), 0644)
			if err != nil {
				t.Fatalf("failed to write file: %s", err)
			}

			err = afero.WriteFile(fs, filepath.Join(dirname, ".terraform.lock.hcl"), []byte(tc.lockfile), 0644)
			if err != nil {
				t.Fatalf("failed to write file: %s", err)
			}

			o := Option{
				updateType: "lock",
				platforms:  platforms,
			}
			gc, err := NewGlobalContext(fs, o)
			if err != nil {
				t.Fatalf("failed to new global context: %s", err)
			}

			// Create a mock index for testing
			mockIndex := lock.NewMockIndex(pvs)

			// Create a LockUpdater with the tfregistryConfig
			u, err := NewLockUpdater(platforms, tc.tfregistryConfig)

			// Replace the index with our mock for testing
			lu := u.(*LockUpdater)
			lu.index = mockIndex
			if err != nil {
				t.Fatalf("failed to new LockUpdater: %s", err)
			}

			f, diags := hclwrite.ParseConfig([]byte(tc.lockfile), ".terraform.lock.hcl", hcl.Pos{Line: 1, Column: 1})
			if diags.HasErrors() {
				t.Fatalf("unexpected diagnostics: %s", diags)
			}

			mc, err := NewModuleContext(dirname, gc)
			if err != nil {
				t.Fatalf("failed to new module context: %s", err)
			}

			err = u.Update(context.Background(), mc, ".terraform.lock.hcl", f)
			if tc.ok && err != nil {
				t.Errorf("failed to call Update: err = %s", err)
			}

			got := string(hclwrite.Format(f.BuildTokens(nil).Bytes()))

			if !tc.ok && err == nil {
				t.Errorf("expect to fail, but success: got = %s", got)
			}

			if diff := cmp.Diff(got, tc.want); diff != "" {
				t.Errorf("got: %s, want = %s, diff = %s", got, tc.want, diff)
			}
		})
	}
}
