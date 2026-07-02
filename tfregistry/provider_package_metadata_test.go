package tfregistry

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"testing"
)

func TestProviderPackageMetadata(t *testing.T) {
	cases := []struct {
		desc string
		req  *ProviderPackageMetadataRequest
		ok   bool
		code int
		res  string
		want *ProviderPackageMetadataResponse
	}{
		{
			desc: "simple",
			req: &ProviderPackageMetadataRequest{
				Namespace: "hashicorp",
				Type:      "null",
				Version:   "3.2.1",
				OS:        "darwin",
				Arch:      "arm64",
			},
			ok:   true,
			code: 200,
			res:  mockProviderPackageMetadataResponse,
			want: &ProviderPackageMetadataResponse{
				Filename:    "terraform-provider-null_3.2.1_darwin_arm64.zip",
				DownloadURL: "https://releases.hashicorp.com/terraform-provider-null/3.2.1/terraform-provider-null_3.2.1_darwin_arm64.zip",
				SHASum:      "e4453fbebf90c53ca3323a92e7ca0f9961427d2f0ce0d2b65523cc04d5d999c2",
				SHASumsURL:  "https://releases.hashicorp.com/terraform-provider-null/3.2.1/terraform-provider-null_3.2.1_SHA256SUMS",
			},
		},
		{
			desc: "not found",
			req: &ProviderPackageMetadataRequest{
				Namespace: "hoge",
				Type:      "piyo",
				Version:   "3.2.1",
				OS:        "darwin",
				Arch:      "arm64",
			},
			ok:   false,
			code: 404,
			res:  `{"errors":["Not Found"]}`,
			want: nil,
		},
		{
			desc: "invalid request (Namespace)",
			req: &ProviderPackageMetadataRequest{
				Namespace: "",
				Type:      "piyo",
				Version:   "3.2.1",
				OS:        "darwin",
				Arch:      "arm64",
			},
			ok:   false,
			code: 0,
			res:  "",
			want: nil,
		},
		{
			desc: "invalid request (Type)",
			req: &ProviderPackageMetadataRequest{
				Namespace: "hoge",
				Type:      "",
				Version:   "3.2.1",
				OS:        "darwin",
				Arch:      "arm64",
			},
			ok:   false,
			code: 0,
			res:  "",
			want: nil,
		},
		{
			desc: "invalid request (Version)",
			req: &ProviderPackageMetadataRequest{
				Namespace: "hoge",
				Type:      "piyo",
				Version:   "",
				OS:        "darwin",
				Arch:      "arm64",
			},
			ok:   false,
			code: 0,
			res:  "",
			want: nil,
		},
		{
			desc: "invalid request (OS)",
			req: &ProviderPackageMetadataRequest{
				Namespace: "hoge",
				Type:      "piyo",
				Version:   "3.2.1",
				OS:        "",
				Arch:      "arm64",
			},
			ok:   false,
			code: 0,
			res:  "",
			want: nil,
		},
		{
			desc: "invalid request (Arch)",
			req: &ProviderPackageMetadataRequest{
				Namespace: "hoge",
				Type:      "piyo",
				Version:   "3.2.1",
				OS:        "darwin",
				Arch:      "",
			},
			ok:   false,
			code: 0,
			res:  "",
			want: nil,
		},
		{
			desc: "with hashes",
			req: &ProviderPackageMetadataRequest{
				Namespace: "hashicorp",
				Type:      "null",
				Version:   "3.2.1",
				OS:        "darwin",
				Arch:      "arm64",
			},
			ok:   true,
			code: 200,
			res:  mockProviderPackageMetadataResponseWithHashes,
			want: &ProviderPackageMetadataResponse{
				Filename:    "terraform-provider-null_3.2.1_darwin_arm64.zip",
				DownloadURL: "https://github.com/opentofu/terraform-provider-null/releases/download/v3.2.1/terraform-provider-null_3.2.1_darwin_arm64.zip",
				SHASum:      "5ce03460813954cbebc9f9ad5befbe364d9dc67acb08869f67c1aa634fbf6d6c",
				SHASumsURL:  "https://github.com/opentofu/terraform-provider-null/releases/download/v3.2.1/terraform-provider-null_3.2.1_SHA256SUMS",
				Packages: map[string]Package{
					"darwin_amd64": {
						Hashes: []string{
							"zh:e25265d4e87821d18dc9653a0ce01978a1ae5d363bc01dd273454db1aa0309c7",
							"h1:BdIx478D4wpFgFeaS+5Bdve1HIlt3Yhsr5hAbUs2rRg=",
						},
					},
					"darwin_arm64": {
						Hashes: []string{
							"zh:5ce03460813954cbebc9f9ad5befbe364d9dc67acb08869f67c1aa634fbf6d6c",
							"h1:+JAon/4CyriC/c7c77NjJalKrKx6gwwQ7L7rVABWMtA=",
						},
					},
					"linux_386": {
						Hashes: []string{
							"zh:c38c9a295cfae9fb6372523c34b9466cd439d5e2c909b56a788960d387c24320",
							"h1:Oio8EVe+5LQF7cd7IYcIrXkyKy8nCrBF/X8DQUAgayQ=",
						},
					},
					"linux_amd64": {
						Hashes: []string{
							"zh:40335019c11e5bdb3780301da5197cbc756b4b5ac3d699c52583f1d34e963176",
							"h1:uQv2oPjJv+ue8bPrVp+So2hHd90UTssnCNajTW554Cw=",
						},
					},
					"linux_arm": {
						Hashes: []string{
							"zh:42356e687656fc8ec1f230f786f830f344e64419552ec483e2bc79bd4b7cf1e8",
							"h1:+s8zqab9SoQhfWDJVRtv9b2pJRyQlZu88NHK0X4bODQ=",
						},
					},
					"linux_arm64": {
						Hashes: []string{
							"zh:91a76f371815a130735c8fcb6196804d878aebcc67b4c3b73571d2063336ffd8",
							"h1:9WdPl8ujc45POfiiCzcX60bljY744neqTGlK24VMhgk=",
						},
					},
					"windows_386": {
						Hashes: []string{
							"zh:658ea3e3e7ecc964bdbd08ecde63f3d79f298bab9922b29a6526ba941a4d403a",
							"h1:K7QNZz/LCsVhvHLA8orqjfVr+qYh4QAPhMi+n9Es6js=",
						},
					},
					"windows_amd64": {
						Hashes: []string{
							"zh:80fd03335f793dc54302dd53da98c91fd94f182bcacf13457bed1a99ecffbc1a",
							"h1:uvf7c5e8N7A0k9dqF2pLbbbPTAHIwUoM885CHCS717E=",
						},
					},
					"windows_arm": {
						Hashes: []string{
							"zh:c146fc0291b7f6284fe4d54ce6aaece6957e9acc93fc572dd505dfd8bcad3e6c",
							"h1:8sG3Tdnsk1STHIKaFORkb9EKaR6eNkzX1MCqWZLo6uY=",
						},
					},
					"windows_arm64": {
						Hashes: []string{
							"zh:68c06703bc57f9c882bfedda6f3047775f0d367093d00efb040800c798b8a613",
							"h1:ckcDeFlUdHEPosRUSxqyCzVdZLh9mrM4ebhygf6c3SA=",
						},
					},
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			mux, mockServerURL := newMockServer()
			client := newTestClient(mockServerURL)
			subPath := fmt.Sprintf("%s%s/%s/%s/download/%s/%s", providerV1Service, tc.req.Namespace, tc.req.Type, tc.req.Version, tc.req.OS, tc.req.Arch)
			mux.HandleFunc(subPath, func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.code)
				fmt.Fprint(w, tc.res)
			})

			got, err := client.ProviderPackageMetadata(context.Background(), tc.req)

			if tc.ok && err != nil {
				t.Fatalf("failed to call ProviderPackageMetadata: err = %s, req = %#v", err, tc.req)
			}

			if !tc.ok && err == nil {
				t.Fatalf("expected to fail, but success: req = %#v, got = %#v", tc.req, got)
			}

			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got=%#v, but want=%#v", got, tc.want)
			}
		})
	}
}

const mockProviderPackageMetadataResponse = `{
  "protocols": [
    "5.0"
  ],
  "os": "darwin",
  "arch": "arm64",
  "filename": "terraform-provider-null_3.2.1_darwin_arm64.zip",
  "download_url": "https://releases.hashicorp.com/terraform-provider-null/3.2.1/terraform-provider-null_3.2.1_darwin_arm64.zip",
  "shasums_url": "https://releases.hashicorp.com/terraform-provider-null/3.2.1/terraform-provider-null_3.2.1_SHA256SUMS",
  "shasums_signature_url": "https://releases.hashicorp.com/terraform-provider-null/3.2.1/terraform-provider-null_3.2.1_SHA256SUMS.72D7468F.sig",
  "shasum": "e4453fbebf90c53ca3323a92e7ca0f9961427d2f0ce0d2b65523cc04d5d999c2",
  "signing_keys": {
    "gpg_public_keys": [
      {
        "key_id": "34365D9472D7468F",
        "ascii_armor": "-----BEGIN PGP PUBLIC KEY BLOCK-----\n\n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\nZF5q4h4I33PSGDdSvGXn9UMY5Isjpg==\n=7pIB\n-----END PGP PUBLIC KEY BLOCK-----",
        "trust_signature": "",
        "source": "HashiCorp",
        "source_url": "https://www.hashicorp.com/security.html"
      }
    ]
  }
}
`

const mockProviderPackageMetadataResponseWithHashes = `{
  "protocols": [
    "5.0"
  ],
  "os": "darwin",
  "arch": "arm64",
  "filename": "terraform-provider-null_3.2.1_darwin_arm64.zip",
  "download_url": "https://github.com/opentofu/terraform-provider-null/releases/download/v3.2.1/terraform-provider-null_3.2.1_darwin_arm64.zip",
  "shasums_url": "https://github.com/opentofu/terraform-provider-null/releases/download/v3.2.1/terraform-provider-null_3.2.1_SHA256SUMS",
  "shasums_signature_url": "https://github.com/opentofu/terraform-provider-null/releases/download/v3.2.1/terraform-provider-null_3.2.1_SHA256SUMS.sig",
  "shasum": "5ce03460813954cbebc9f9ad5befbe364d9dc67acb08869f67c1aa634fbf6d6c",
  "signing_keys": {
    "gpg_public_keys": [
      {
        "ascii_armor": "-----BEGIN PGP PUBLIC KEY BLOCK-----\n\nxsFNBGVUyIwBEADPg6jUJm5liMTiDndyprnwXQ23GdyQm/kW9MFOhYDRksmmbsz0\nDCfqntFpuoKxPXzA+JTrZlWZONtU+leZjIOlAVZiz0rwz5EJq7uIrkueWtUk6AYk\nBLN+zMtbui0z3HCPVNnR5BlVNyXQeW3jlrQtzuKevjZWzI0gbQGgEKNpj+lfyRFu\n6q3u/T0o3p/6bOOlQHwCMtnFlWpjr6f/J2EdUVO/6NYHQzImPj4LINXF/+eqo7v6\nsvFtaVTtREG2V2V7We7bu/cJ+NgJYH7ro7UhB1RQH2k09NdpSCt9F60PVERnORpx\nGBkM/VKZzgMSzRvdpxUWwrLxfAxinu5ddbBm3y0bzaU80OT3i1qrWIqW73fmdGHQ\n71gbJxRrroyLMWehjcJ/9WJDxkHqsfPKqBifYsp6/J9npczDfSU+zYBVGpR73a4E\ndbeIRWqwbH0LWhlbi1IM5aFDaZMFNkY+AWyP+OHn8Kehu6DOIh1AVM7v7vLxaX9h\nt1jVJbswjvPFYquv1DvUdc7VP2QHz3xctQS1GZJQ1ekcgTv9rRYXUOOwknInjtkM\n9kQDtyBkVLcEc8ha3Cfh6PJscIP5VHwaNMgAPr9tsl3xqdz56l5UPjFSFuel98jS\nBqn83VrT0uKwM0PnDVHd/7q8+Dg1EtOggMwZ830KORFNdjfv6ydsBvl7fwARAQAB\nzUpPcGVuVG9mdSAoVGhpcyBrZXkgaXMgdXNlZCB0byBzaWduIG9wZW50b2Z1IHBy\nb3ZpZGVycykgPGNvcmVAb3BlbnRvZnUub3JnPsLBjAQTAQgAQQUCZVTIjAkQDArz\nE+X9n4AWIQTj5uQ9hMuFLq2wBR0MCvMT5f2fgAIbAwIeAQIZAQMLCQcCFQgDFgAC\nBScJAgcCAABwAg/1HZnTvPHZDWf5OluYOaQ7ADX/oyjUO85VNUmKhmBZkLr5mTqr\nLO72k9fg+101hbggbhtK431z3Ca6ZqDAG/3DBi0BC1ag0rw83TEApkPGYnfX1DWS\n1ZvyH1PkV0aqCkXAtMrte2PlUiieaKAsiYOIXqfZwszd07gch14wxMOw1B6Au/Xz\nNrv2omnWSgGIyR6WOsG4QQ8R5AMVz3K8Ftzl6520wBgtr3osA3uM/xconnGVukMn\n9NLQqKx5oeaJwONZpyZL5bg2ke9MVZM2+bG30UGZKoxrzOtQ//OTOYlhPCqm1ffR\nhYrUytwsWzDnJvXJF1QhnDu8whP3tSrcHyKxYZ9xUNzeu2AmjYfvkKHSdK2DFmOf\nDafaRs3c1VYnC7J7aRi6kVF/t+vWeOEVpPylyK7vSbPFc6XVoQrsE07hbN/BjWjm\ns8voK5U6oJRgEugXtSQKFypfOq8R99nXwbMHdhqY8aGyOCj++cuvRCUBDZAQqPEW\nAuD0X7+9Trnfin47MK+n18wsTAL4w6PJhtCrwK4e0cVuQ5u4M/PMid5W6hEA27PX\nx506Jpe8iRmcIP/cCR6pvhgOUMC36bIkAqZ5dJ545kDQju0lf8gLdVIQpig45udn\nZM2KgyApGqhsS7yCUrbLDrtNmQ31TSYdKc8IU+/jXkfy2RYbZ+wNgfloKM7BTQRl\nVMiMARAAwRZUyMIc5TNbcFg3WGKxhaNC9hDZ4zBfXlb5jONzZOx3rDi2lD4UQOH+\nNpG7CF98co//kryS/4AsDdp2jzhh+VMgyx6KJIhSkBP6kqhriy9eWRmgfrnLbUf4\n6kkTkzLVkjYnMNeyHt+mi9I7EKtsDuF/EvjlwF5E81+DEOteCO/un/Qt1q3e1Slf\nvTpLkPvr1FiQ3VqzaBeBBI3MAMb/ycwL6hQE1l4Lg34T43Zu+9zkE1uzvjeNIlIW\nucjB4q1htEjJl2CLAv+8cGHdmCcV2ZO3WM8M9Omq1CE7jhak4NE/YuGylJYCBd+B\nS7tuDPDu6+o4Nx+axxcwMvgyfr07FteEr1Lopaw2ci8b/xzQie/gkI0CByQMwD5V\ngnJpiMBnjP4d6UF6HEVldCQ7a3T1T80bKj5JjtFbR9P85Qntuheqn3Pge89YexMc\nE/00VA3blrj+GeYpO9ZGFu7DR/x4sjnTEhfjXEoLv1C4AdgGHCIjW9wU6HkcWnla\nX7akKlwIWEUP/BFLkcWPpmUrtClhWx9wq1GHFvKAN/qp//VWnv4IfRU6RjmVPOWB\nefvTu/cpsfBHLyp15goOYPboahIdTUTNQIXh4Vid7E1NoKnWZUMu50n3/zAbjSds\nmNmifi4g01MYJ3TVoU2Q01P7NiD3IRmaw72nLmf9cM9/7QMdGn0AEQEAAcLBdgQY\nAQgAKgUCZVTIjAkQDArzE+X9n4AWIQTj5uQ9hMuFLq2wBR0MCvMT5f2fgAIbDAAA\nSUoP/2ExsUoGbxjuZ76QUnYtfzDoz+o218UWd3gZCsBQ6/hGam5kMq+EUEabF3lV\n7QLDyn/1v5sqrkmYg0u5cfjtY3oimCPvr6E0WTuqMIwYl0fdlkmdNttDpMqvCazq\nbzLK5dDVWbh/EYTiEN1xKXM6rlAquYv8I16uWL8QHanMb6yexNmDYhC4fXWqCi+s\n5sXxWrPrd+fGz8CR/fEYahPXj8uY6dwN9DlWyek9QtKW2PsqrkBn5vCOm2IyZW6d\nt/Kn70tYtxMxJND2otk47mpG/Fv3sYK2bTGJ+k/5+E5IrjWqIX2lVB3G1+TCoZ5s\ncc16zls32mOlRh81fTAqcwkDFxICxcOeNHGLt3N+UvoPSUafYKD96rn5mWFao4xb\ncFniaYv2PdqH8HDjvXZXqHypRMXvYMbXXOgydLL+tSUSBpMTd4afjq8x2gNSWOEL\nI1jT5FWbKTKan0ycKi37bSqGHhDjlg4HRGvC3IK0EuVjdX3r+8uIVgFbqLwNhXk4\nGAIL03vl689TQ7/oPW75XCQIevFai0kcJPl6qIRvi9/S/v5EPRy9UDCGY/MPmc5f\nH1an0ebU4I4TlYfBoEUkYYqBDxvxWW0I/Q01rDebcd6mrGw8lW1EiNZlClLwx9Bv\n/+MNnIT9m1f8KeqmweoAgbIQRUI7EkJSzxYN4DNuy2XoKmF9\n=VhyH\n-----END PGP PUBLIC KEY BLOCK-----",
        "key_id": "0C0AF313E5FD9F80"
      }
    ]
  },
  "packages": {
    "darwin_amd64": {
      "hashes": [
        "zh:e25265d4e87821d18dc9653a0ce01978a1ae5d363bc01dd273454db1aa0309c7",
        "h1:BdIx478D4wpFgFeaS+5Bdve1HIlt3Yhsr5hAbUs2rRg="
      ],
      "package_size": 4945518
    },
    "darwin_arm64": {
      "hashes": [
        "zh:5ce03460813954cbebc9f9ad5befbe364d9dc67acb08869f67c1aa634fbf6d6c",
        "h1:+JAon/4CyriC/c7c77NjJalKrKx6gwwQ7L7rVABWMtA="
      ],
      "package_size": 4781117
    },
    "linux_386": {
      "hashes": [
        "zh:c38c9a295cfae9fb6372523c34b9466cd439d5e2c909b56a788960d387c24320",
        "h1:Oio8EVe+5LQF7cd7IYcIrXkyKy8nCrBF/X8DQUAgayQ="
      ],
      "package_size": 4543480
    },
    "linux_amd64": {
      "hashes": [
        "zh:40335019c11e5bdb3780301da5197cbc756b4b5ac3d699c52583f1d34e963176",
        "h1:uQv2oPjJv+ue8bPrVp+So2hHd90UTssnCNajTW554Cw="
      ],
      "package_size": 4750928
    },
    "linux_arm": {
      "hashes": [
        "zh:42356e687656fc8ec1f230f786f830f344e64419552ec483e2bc79bd4b7cf1e8",
        "h1:+s8zqab9SoQhfWDJVRtv9b2pJRyQlZu88NHK0X4bODQ="
      ],
      "package_size": 4481884
    },
    "linux_arm64": {
      "hashes": [
        "zh:91a76f371815a130735c8fcb6196804d878aebcc67b4c3b73571d2063336ffd8",
        "h1:9WdPl8ujc45POfiiCzcX60bljY744neqTGlK24VMhgk="
      ],
      "package_size": 4368837
    },
    "windows_386": {
      "hashes": [
        "zh:658ea3e3e7ecc964bdbd08ecde63f3d79f298bab9922b29a6526ba941a4d403a",
        "h1:K7QNZz/LCsVhvHLA8orqjfVr+qYh4QAPhMi+n9Es6js="
      ],
      "package_size": 4669934
    },
    "windows_amd64": {
      "hashes": [
        "zh:80fd03335f793dc54302dd53da98c91fd94f182bcacf13457bed1a99ecffbc1a",
        "h1:uvf7c5e8N7A0k9dqF2pLbbbPTAHIwUoM885CHCS717E="
      ],
      "package_size": 4763231
    },
    "windows_arm": {
      "hashes": [
        "zh:c146fc0291b7f6284fe4d54ce6aaece6957e9acc93fc572dd505dfd8bcad3e6c",
        "h1:8sG3Tdnsk1STHIKaFORkb9EKaR6eNkzX1MCqWZLo6uY="
      ],
      "package_size": 4538273
    },
    "windows_arm64": {
      "hashes": [
        "zh:68c06703bc57f9c882bfedda6f3047775f0d367093d00efb040800c798b8a613",
        "h1:ckcDeFlUdHEPosRUSxqyCzVdZLh9mrM4ebhygf6c3SA="
      ],
      "package_size": 4381068
    }
  }
}
`
