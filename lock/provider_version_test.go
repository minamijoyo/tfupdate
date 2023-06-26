package lock

import (
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/go-cmp/cmp"
)

func TestProviderVersionMerge(t *testing.T) {
	cases := []struct {
		desc string
		pv   *ProviderVersion
		rhs  *ProviderVersion
		want *ProviderVersion
		ok   bool
	}{
		{
			desc: "simple",
			pv: &ProviderVersion{
				address:   "minamijoyo/dummy",
				version:   "3.2.1",
				platforms: []string{"darwin_arm64", "darwin_amd64"},
				h1Hashes: map[string]string{
					"terraform-provider-dummy_3.2.1_darwin_arm64.zip": "h1:3323G20HW9PA9ONrL6CdQCdCFe6y94kXeOTprq+Zu+w=",
					"terraform-provider-dummy_3.2.1_darwin_amd64.zip": "h1:63My0EuWIYHWVwWOxmxWwgrfx+58Tz+nTduelaCCAfs=",
				},
				zhHashes: map[string]string{
					"terraform-provider-dummy_3.2.1_darwin_arm64.zip":  "zh:5622a0fd03420ed1fa83a1a6e90b65fbe34bc74c251b3b47048f14217e93b086",
					"terraform-provider-dummy_3.2.1_darwin_amd64.zip":  "zh:fc5bbdd0a1bd6715b9afddf3aba6acc494425d77015c19579b9a9fa950e532b2",
					"terraform-provider-dummy_3.2.1_linux_amd64.zip":   "zh:c5f0a44e3a3795cb3ee0abb0076097c738294c241f74c145dfb50f2b9fd71fd2",
					"terraform-provider-dummy_3.2.1_windows_amd64.zip": "zh:8b75ff41191a7fe6c5d9129ed19a01eacde5a3797b48b738eefa21f5330c081e",
				},
			},
			rhs: &ProviderVersion{
				address:   "minamijoyo/dummy",
				version:   "3.2.1",
				platforms: []string{"linux_amd64"},
				h1Hashes: map[string]string{
					"terraform-provider-dummy_3.2.1_linux_amd64.zip": "h1:2zotrPRAjGZZMkjJGBGLnIbG+sqhQN30sbwqSDECQFQ=",
				},
				zhHashes: map[string]string{
					"terraform-provider-dummy_3.2.1_darwin_arm64.zip":  "zh:5622a0fd03420ed1fa83a1a6e90b65fbe34bc74c251b3b47048f14217e93b086",
					"terraform-provider-dummy_3.2.1_darwin_amd64.zip":  "zh:fc5bbdd0a1bd6715b9afddf3aba6acc494425d77015c19579b9a9fa950e532b2",
					"terraform-provider-dummy_3.2.1_linux_amd64.zip":   "zh:c5f0a44e3a3795cb3ee0abb0076097c738294c241f74c145dfb50f2b9fd71fd2",
					"terraform-provider-dummy_3.2.1_windows_amd64.zip": "zh:8b75ff41191a7fe6c5d9129ed19a01eacde5a3797b48b738eefa21f5330c081e",
				},
			},
			want: &ProviderVersion{
				address:   "minamijoyo/dummy",
				version:   "3.2.1",
				platforms: []string{"darwin_arm64", "darwin_amd64", "linux_amd64"},
				h1Hashes: map[string]string{
					"terraform-provider-dummy_3.2.1_darwin_arm64.zip": "h1:3323G20HW9PA9ONrL6CdQCdCFe6y94kXeOTprq+Zu+w=",
					"terraform-provider-dummy_3.2.1_darwin_amd64.zip": "h1:63My0EuWIYHWVwWOxmxWwgrfx+58Tz+nTduelaCCAfs=",
					"terraform-provider-dummy_3.2.1_linux_amd64.zip":  "h1:2zotrPRAjGZZMkjJGBGLnIbG+sqhQN30sbwqSDECQFQ=",
				},
				zhHashes: map[string]string{
					"terraform-provider-dummy_3.2.1_darwin_arm64.zip":  "zh:5622a0fd03420ed1fa83a1a6e90b65fbe34bc74c251b3b47048f14217e93b086",
					"terraform-provider-dummy_3.2.1_darwin_amd64.zip":  "zh:fc5bbdd0a1bd6715b9afddf3aba6acc494425d77015c19579b9a9fa950e532b2",
					"terraform-provider-dummy_3.2.1_linux_amd64.zip":   "zh:c5f0a44e3a3795cb3ee0abb0076097c738294c241f74c145dfb50f2b9fd71fd2",
					"terraform-provider-dummy_3.2.1_windows_amd64.zip": "zh:8b75ff41191a7fe6c5d9129ed19a01eacde5a3797b48b738eefa21f5330c081e",
				},
			},
			ok: true,
		},
		{
			desc: "merge to empty",
			pv:   newEmptyProviderVersion("minamijoyo/dummy", "3.2.1"),
			rhs: &ProviderVersion{
				address:   "minamijoyo/dummy",
				version:   "3.2.1",
				platforms: []string{"linux_amd64"},
				h1Hashes: map[string]string{
					"terraform-provider-dummy_3.2.1_linux_amd64.zip": "h1:2zotrPRAjGZZMkjJGBGLnIbG+sqhQN30sbwqSDECQFQ=",
				},
				zhHashes: map[string]string{
					"terraform-provider-dummy_3.2.1_darwin_arm64.zip":  "zh:5622a0fd03420ed1fa83a1a6e90b65fbe34bc74c251b3b47048f14217e93b086",
					"terraform-provider-dummy_3.2.1_darwin_amd64.zip":  "zh:fc5bbdd0a1bd6715b9afddf3aba6acc494425d77015c19579b9a9fa950e532b2",
					"terraform-provider-dummy_3.2.1_linux_amd64.zip":   "zh:c5f0a44e3a3795cb3ee0abb0076097c738294c241f74c145dfb50f2b9fd71fd2",
					"terraform-provider-dummy_3.2.1_windows_amd64.zip": "zh:8b75ff41191a7fe6c5d9129ed19a01eacde5a3797b48b738eefa21f5330c081e",
				},
			},
			want: &ProviderVersion{
				address:   "minamijoyo/dummy",
				version:   "3.2.1",
				platforms: []string{"linux_amd64"},
				h1Hashes: map[string]string{
					"terraform-provider-dummy_3.2.1_linux_amd64.zip": "h1:2zotrPRAjGZZMkjJGBGLnIbG+sqhQN30sbwqSDECQFQ=",
				},
				zhHashes: map[string]string{
					"terraform-provider-dummy_3.2.1_darwin_arm64.zip":  "zh:5622a0fd03420ed1fa83a1a6e90b65fbe34bc74c251b3b47048f14217e93b086",
					"terraform-provider-dummy_3.2.1_darwin_amd64.zip":  "zh:fc5bbdd0a1bd6715b9afddf3aba6acc494425d77015c19579b9a9fa950e532b2",
					"terraform-provider-dummy_3.2.1_linux_amd64.zip":   "zh:c5f0a44e3a3795cb3ee0abb0076097c738294c241f74c145dfb50f2b9fd71fd2",
					"terraform-provider-dummy_3.2.1_windows_amd64.zip": "zh:8b75ff41191a7fe6c5d9129ed19a01eacde5a3797b48b738eefa21f5330c081e",
				},
			},
			ok: true,
		},
		{
			desc: "address mismatch",
			pv:   newEmptyProviderVersion("minamijoyo/dummy", "3.2.1"),
			rhs:  newEmptyProviderVersion("minamijoyo/foo", "3.2.1"),
			want: newEmptyProviderVersion("minamijoyo/dummy", "3.2.1"),
			ok:   false,
		},
		{
			desc: "version mismatch",
			pv:   newEmptyProviderVersion("minamijoyo/dummy", "3.2.1"),
			rhs:  newEmptyProviderVersion("minamijoyo/dummy", "3.2.0"),
			want: newEmptyProviderVersion("minamijoyo/dummy", "3.2.1"),
			ok:   false,
		},
		{
			desc: "zh mismatch",
			pv: &ProviderVersion{
				address:   "minamijoyo/dummy",
				version:   "3.2.1",
				platforms: []string{"darwin_arm64", "darwin_amd64"},
				h1Hashes: map[string]string{
					"terraform-provider-dummy_3.2.1_darwin_arm64.zip": "h1:3323G20HW9PA9ONrL6CdQCdCFe6y94kXeOTprq+Zu+w=",
					"terraform-provider-dummy_3.2.1_darwin_amd64.zip": "h1:63My0EuWIYHWVwWOxmxWwgrfx+58Tz+nTduelaCCAfs=",
				},
				zhHashes: map[string]string{
					"terraform-provider-dummy_3.2.1_darwin_arm64.zip":  "zh:5622a0fd03420ed1fa83a1a6e90b65fbe34bc74c251b3b47048f14217e93b086",
					"terraform-provider-dummy_3.2.1_darwin_amd64.zip":  "zh:fc5bbdd0a1bd6715b9afddf3aba6acc494425d77015c19579b9a9fa950e532b2",
					"terraform-provider-dummy_3.2.1_linux_amd64.zip":   "zh:c5f0a44e3a3795cb3ee0abb0076097c738294c241f74c145dfb50f2b9fd71fd2",
					"terraform-provider-dummy_3.2.1_windows_amd64.zip": "zh:8b75ff41191a7fe6c5d9129ed19a01eacde5a3797b48b738eefa21f5330c081e",
				},
			},
			rhs: &ProviderVersion{
				address:   "minamijoyo/dummy",
				version:   "3.2.1",
				platforms: []string{"linux_amd64"},
				h1Hashes: map[string]string{
					"terraform-provider-dummy_3.2.1_linux_amd64.zip": "h1:2zotrPRAjGZZMkjJGBGLnIbG+sqhQN30sbwqSDECQFQ=",
				},
				zhHashes: map[string]string{
					"terraform-provider-dummy_3.2.1_darwin_arm64.zip":  "zh:5622a0fd03420ed1fa83a1a6e90b65fbe34bc74c251b3b47048f14217e93b086",
					"terraform-provider-dummy_3.2.1_darwin_amd64.zip":  "zh:fc5bbdd0a1bd6715b9afddf3aba6acc494425d77015c19579b9a9fa950e532b2",
					"terraform-provider-dummy_3.2.1_linux_amd64.zip":   "zh:c5f0a44e3a3795cb3ee0abb0076097c738294c241f74c145dfb50f2b9fd71fd2",
					"terraform-provider-dummy_3.2.1_windows_amd64.zip": "zh:0000000000000000000000000000000000000000000000000000000000000000",
				},
			},
			want: &ProviderVersion{
				address:   "minamijoyo/dummy",
				version:   "3.2.1",
				platforms: []string{"darwin_arm64", "darwin_amd64", "linux_amd64"},
				h1Hashes: map[string]string{
					"terraform-provider-dummy_3.2.1_darwin_arm64.zip": "h1:3323G20HW9PA9ONrL6CdQCdCFe6y94kXeOTprq+Zu+w=",
					"terraform-provider-dummy_3.2.1_darwin_amd64.zip": "h1:63My0EuWIYHWVwWOxmxWwgrfx+58Tz+nTduelaCCAfs=",
					"terraform-provider-dummy_3.2.1_linux_amd64.zip":  "h1:2zotrPRAjGZZMkjJGBGLnIbG+sqhQN30sbwqSDECQFQ=",
				},
				zhHashes: map[string]string{
					"terraform-provider-dummy_3.2.1_darwin_arm64.zip":  "zh:5622a0fd03420ed1fa83a1a6e90b65fbe34bc74c251b3b47048f14217e93b086",
					"terraform-provider-dummy_3.2.1_darwin_amd64.zip":  "zh:fc5bbdd0a1bd6715b9afddf3aba6acc494425d77015c19579b9a9fa950e532b2",
					"terraform-provider-dummy_3.2.1_linux_amd64.zip":   "zh:c5f0a44e3a3795cb3ee0abb0076097c738294c241f74c145dfb50f2b9fd71fd2",
					"terraform-provider-dummy_3.2.1_windows_amd64.zip": "zh:8b75ff41191a7fe6c5d9129ed19a01eacde5a3797b48b738eefa21f5330c081e",
				},
			},
			ok: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {

			err := tc.pv.Merge(tc.rhs)

			if tc.ok && err != nil {
				t.Fatalf("failed to call Merge: err = %s", err)
			}

			if !tc.ok && err == nil {
				t.Fatalf("expected to fail, but success: got = %s", spew.Sdump(tc.pv))
			}

			if diff := cmp.Diff(tc.pv, tc.want, cmp.AllowUnexported(ProviderVersion{})); diff != "" {
				t.Errorf("got: %s, want = %s, diff = %s", spew.Sdump(tc.pv), spew.Sdump(tc.want), diff)
			}
		})
	}
}

func TestProviderVersionAllHashes(t *testing.T) {
	cases := []struct {
		desc string
		pv   *ProviderVersion
		want []string
	}{
		{
			desc: "simple",
			pv: &ProviderVersion{
				address:   "minamijoyo/dummy",
				version:   "3.2.1",
				platforms: []string{"darwin_arm64", "darwin_amd64", "linux_amd64", "windows_amd64"},
				h1Hashes: map[string]string{
					"terraform-provider-dummy_3.2.1_darwin_arm64.zip": "h1:3323G20HW9PA9ONrL6CdQCdCFe6y94kXeOTprq+Zu+w=",
					"terraform-provider-dummy_3.2.1_darwin_amd64.zip": "h1:63My0EuWIYHWVwWOxmxWwgrfx+58Tz+nTduelaCCAfs=",
					"terraform-provider-dummy_3.2.1_linux_amd64.zip":  "h1:2zotrPRAjGZZMkjJGBGLnIbG+sqhQN30sbwqSDECQFQ=",
				},
				zhHashes: map[string]string{
					"terraform-provider-dummy_3.2.1_darwin_arm64.zip":  "zh:5622a0fd03420ed1fa83a1a6e90b65fbe34bc74c251b3b47048f14217e93b086",
					"terraform-provider-dummy_3.2.1_darwin_amd64.zip":  "zh:fc5bbdd0a1bd6715b9afddf3aba6acc494425d77015c19579b9a9fa950e532b2",
					"terraform-provider-dummy_3.2.1_linux_amd64.zip":   "zh:c5f0a44e3a3795cb3ee0abb0076097c738294c241f74c145dfb50f2b9fd71fd2",
					"terraform-provider-dummy_3.2.1_windows_amd64.zip": "zh:8b75ff41191a7fe6c5d9129ed19a01eacde5a3797b48b738eefa21f5330c081e",
				},
			},
			want: []string{
				"h1:2zotrPRAjGZZMkjJGBGLnIbG+sqhQN30sbwqSDECQFQ=",
				"h1:3323G20HW9PA9ONrL6CdQCdCFe6y94kXeOTprq+Zu+w=",
				"h1:63My0EuWIYHWVwWOxmxWwgrfx+58Tz+nTduelaCCAfs=",
				"zh:5622a0fd03420ed1fa83a1a6e90b65fbe34bc74c251b3b47048f14217e93b086",
				"zh:8b75ff41191a7fe6c5d9129ed19a01eacde5a3797b48b738eefa21f5330c081e",
				"zh:c5f0a44e3a3795cb3ee0abb0076097c738294c241f74c145dfb50f2b9fd71fd2",
				"zh:fc5bbdd0a1bd6715b9afddf3aba6acc494425d77015c19579b9a9fa950e532b2",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {

			got := tc.pv.AllHashes()

			if diff := cmp.Diff(got, tc.want); diff != "" {
				t.Errorf("got: %s, want = %s, diff = %s", spew.Sdump(got), spew.Sdump(tc.want), diff)
			}
		})
	}
}
