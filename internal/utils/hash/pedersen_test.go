package hash

import (
	"fmt"
	"math/big"
	"strings"
	"testing"

	path "github.com/thrasher-corp/gocryptotrader/internal/testing/utils"
)

func TestPedersen(t *testing.T) {
	rootPath, err := path.RootPathFromCWD()
	if err != nil {
		t.Fatal(err)
	}
	const defaultPedersenConfigsPath = "internal/utils/hash/elliptic_curve_config/"
	loadConfig, err := LoadPedersenConfig(rootPath + "/" + defaultPedersenConfigsPath + strings.ToLower("starkEx") + ".json")
	if err != nil {
		t.Fatal(t, err)
	}

	tests := []struct {
		a, b string
		want string
	}{
		{
			"0x03d937c035c878245caf64531a5756109c53068da139362728feb561405371cb",
			"0x0208a0a10250e382e1e4bbe2880906c2791bf6275695e02fbbc6aeff9cd8b31a",
			"1382171651951541052082654537810074813456022260470662576358627909045455537762",
		},
		{
			"0x58f580910a6ca59b28927c08fe6c43e2e303ca384badc365795fc645d479d45",
			"0x78734f65a067be9bdb39de18434d71e79f7b6466a4b66bbd979ab9e7515fe0b",
			"2962565761002374879415469392216379291665599807391815720833106117558254791559",
		},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("TestHash %d", i), func(t *testing.T) {
			a, ok := big.NewInt(0).SetString(tt.a, 0)
			if !ok {
				t.Errorf("expected no error but got ")
			}
			b, ok := big.NewInt(0).SetString(tt.b, 0)
			if !ok {
				t.Errorf("expected no error but got ")
			}

			if len(tt.want) == 65 && strings.HasPrefix(tt.want, "0x") {
				tt.want = strings.Replace(tt.want, "0x", "0x0", 1)
			}
			and := loadConfig.PedersenHash(a.String(), b.String())
			if !strings.EqualFold(and, tt.want) {
				t.Errorf("TestHash got %s, want %s", and, tt.want)
			}
		})
	}
}
