package tfupdate

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/minamijoyo/tfupdate/lock"
	"github.com/spf13/afero"
)

func TestNewLockUpdater(t *testing.T) {
	index := lock.NewMockIndex([]*lock.ProviderVersion{})
	cases := []struct {
		platforms []string
		index     lock.Index
		want      Updater
		ok        bool
	}{
		{
			platforms: []string{"darwin_arm64", "darwin_amd64", "linux_amd64"},
			index:     index,
			want: &LockUpdater{
				platforms: []string{"darwin_arm64", "darwin_amd64", "linux_amd64"},
				index:     index,
			},
			ok: true,
		},
	}

	for _, tc := range cases {
		got, err := NewLockUpdater(tc.platforms, tc.index)
		if tc.ok && err != nil {
			t.Errorf("NewLockUpdater() with platforms = %#v returns unexpected err: %+v", tc.platforms, err)
		}

		if !tc.ok && err == nil {
			t.Errorf("NewLockUpdater() with platforms = %#v expects to return an error, but no error", tc.platforms)
		}

		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("NewLockUpdater() with platforms = %s returns %#v, but want = %#v", tc.platforms, got, tc.want)
		}
	}
}

func TestUpdateLock(t *testing.T) {
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
		{
			desc: "ignore unsupported constaints",
			src: `
terraform {
  required_version = "1.5.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = ">= 5.3.0"
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
  constraints = ">= 5.3.0"
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
  constraints = ">= 5.3.0"
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

			index := lock.NewMockIndex(pvs)
			u, err := NewLockUpdater(platforms, index)
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
				t.Errorf("faild to call Update: err = %s", err)
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
