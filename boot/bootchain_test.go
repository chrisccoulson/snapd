// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * Copyright (C) 2020 Canonical Ltd
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License version 3 as
 * published by the Free Software Foundation.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

package boot_test

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"

	. "gopkg.in/check.v1"

	"github.com/snapcore/snapd/boot"
	"github.com/snapcore/snapd/bootloader"
	"github.com/snapcore/snapd/dirs"
	"github.com/snapcore/snapd/secboot"
	"github.com/snapcore/snapd/testutil"
)

type bootchainSuite struct {
	testutil.BaseTest

	rootDir string
}

var _ = Suite(&bootchainSuite{})

func (s *bootchainSuite) SetUpTest(c *C) {
	s.BaseTest.SetUpTest(c)
	s.rootDir = c.MkDir()
	s.AddCleanup(func() { dirs.SetRootDir("/") })
	dirs.SetRootDir(s.rootDir)

	c.Assert(os.MkdirAll(filepath.Join(dirs.SnapBootAssetsDir), 0755), IsNil)
}

func (s *bootchainSuite) TestBootAssetsSort(c *C) {
	// by role
	d := []boot.BootAsset{
		{Role: "run", Name: "1ist", Hashes: []string{"b", "c"}},
		{Role: "recovery", Name: "1ist", Hashes: []string{"b", "c"}},
	}
	sort.Sort(boot.ByBootAssetOrder(d))
	c.Check(d, DeepEquals, []boot.BootAsset{
		{Role: "recovery", Name: "1ist", Hashes: []string{"b", "c"}},
		{Role: "run", Name: "1ist", Hashes: []string{"b", "c"}},
	})

	// by name
	d = []boot.BootAsset{
		{Role: "recovery", Name: "shim", Hashes: []string{"d", "e"}},
		{Role: "recovery", Name: "loader", Hashes: []string{"d", "e"}},
	}
	sort.Sort(boot.ByBootAssetOrder(d))
	c.Check(d, DeepEquals, []boot.BootAsset{
		{Role: "recovery", Name: "loader", Hashes: []string{"d", "e"}},
		{Role: "recovery", Name: "shim", Hashes: []string{"d", "e"}},
	})

	// by hash list length
	d = []boot.BootAsset{
		{Role: "run", Name: "1ist", Hashes: []string{"a", "f"}},
		{Role: "run", Name: "1ist", Hashes: []string{"d"}},
	}
	sort.Sort(boot.ByBootAssetOrder(d))
	c.Check(d, DeepEquals, []boot.BootAsset{
		{Role: "run", Name: "1ist", Hashes: []string{"d"}},
		{Role: "run", Name: "1ist", Hashes: []string{"a", "f"}},
	})

	// hash list entries
	d = []boot.BootAsset{
		{Role: "run", Name: "1ist", Hashes: []string{"b", "d"}},
		{Role: "run", Name: "1ist", Hashes: []string{"b", "c"}},
	}
	sort.Sort(boot.ByBootAssetOrder(d))
	c.Check(d, DeepEquals, []boot.BootAsset{
		{Role: "run", Name: "1ist", Hashes: []string{"b", "c"}},
		{Role: "run", Name: "1ist", Hashes: []string{"b", "d"}},
	})

	d = []boot.BootAsset{
		{Role: "run", Name: "loader", Hashes: []string{"z"}},
		{Role: "recovery", Name: "shim", Hashes: []string{"b"}},
		{Role: "run", Name: "loader", Hashes: []string{"c", "d"}},
		{Role: "run", Name: "1oader", Hashes: []string{"d", "e"}},
		{Role: "recovery", Name: "loader", Hashes: []string{"d", "e"}},
		{Role: "run", Name: "0oader", Hashes: []string{"x", "z"}},
	}
	sort.Sort(boot.ByBootAssetOrder(d))
	c.Check(d, DeepEquals, []boot.BootAsset{
		{Role: "recovery", Name: "loader", Hashes: []string{"d", "e"}},
		{Role: "recovery", Name: "shim", Hashes: []string{"b"}},
		{Role: "run", Name: "0oader", Hashes: []string{"x", "z"}},
		{Role: "run", Name: "1oader", Hashes: []string{"d", "e"}},
		{Role: "run", Name: "loader", Hashes: []string{"z"}},
		{Role: "run", Name: "loader", Hashes: []string{"c", "d"}},
	})

	// d is already sorted, sort it again
	sort.Sort(boot.ByBootAssetOrder(d))
	// still the same
	c.Check(d, DeepEquals, []boot.BootAsset{
		{Role: "recovery", Name: "loader", Hashes: []string{"d", "e"}},
		{Role: "recovery", Name: "shim", Hashes: []string{"b"}},
		{Role: "run", Name: "0oader", Hashes: []string{"x", "z"}},
		{Role: "run", Name: "1oader", Hashes: []string{"d", "e"}},
		{Role: "run", Name: "loader", Hashes: []string{"z"}},
		{Role: "run", Name: "loader", Hashes: []string{"c", "d"}},
	})

	// 2 identical entries
	d = []boot.BootAsset{
		{Role: "run", Name: "loader", Hashes: []string{"x", "z"}},
		{Role: "run", Name: "loader", Hashes: []string{"x", "z"}},
	}
	sort.Sort(boot.ByBootAssetOrder(d))
	c.Check(d, DeepEquals, []boot.BootAsset{
		{Role: "run", Name: "loader", Hashes: []string{"x", "z"}},
		{Role: "run", Name: "loader", Hashes: []string{"x", "z"}},
	})

}

func (s *bootchainSuite) TestBootAssetsPredictable(c *C) {
	// by role
	ba := boot.BootAsset{
		Role: "run", Name: "list", Hashes: []string{"b", "a"},
	}
	pred := boot.ToPredictableBootAsset(&ba)
	c.Check(pred, DeepEquals, &boot.BootAsset{
		Role: "run", Name: "list", Hashes: []string{"a", "b"},
	})
	// original structure is not changed
	c.Check(ba, DeepEquals, boot.BootAsset{
		Role: "run", Name: "list", Hashes: []string{"b", "a"},
	})

	// try to make a predictable struct predictable once more
	predAgain := boot.ToPredictableBootAsset(pred)
	c.Check(predAgain, DeepEquals, pred)

	baNil := boot.ToPredictableBootAsset(nil)
	c.Check(baNil, IsNil)
}

func (s *bootchainSuite) TestBootChainMarshalOnlyAssets(c *C) {
	pbNil := boot.ToPredictableBootChain(nil)
	c.Check(pbNil, IsNil)

	bc := &boot.BootChain{
		AssetChain: []boot.BootAsset{
			{Role: "run", Name: "loader", Hashes: []string{"z"}},
			{Role: "recovery", Name: "shim", Hashes: []string{"b"}},
			{Role: "run", Name: "loader", Hashes: []string{"d", "c"}},
			{Role: "run", Name: "1oader", Hashes: []string{"e", "d"}},
			{Role: "recovery", Name: "loader", Hashes: []string{"e", "d"}},
			{Role: "run", Name: "0oader", Hashes: []string{"z", "x"}},
		},
	}

	predictableBc := boot.ToPredictableBootChain(bc)

	c.Check(predictableBc, DeepEquals, &boot.BootChain{
		// assets are sorted
		AssetChain: []boot.BootAsset{
			// hash lists are sorted
			{Role: "recovery", Name: "loader", Hashes: []string{"d", "e"}},
			{Role: "recovery", Name: "shim", Hashes: []string{"b"}},
			{Role: "run", Name: "0oader", Hashes: []string{"x", "z"}},
			{Role: "run", Name: "1oader", Hashes: []string{"d", "e"}},
			{Role: "run", Name: "loader", Hashes: []string{"z"}},
			{Role: "run", Name: "loader", Hashes: []string{"c", "d"}},
		},
	})

	d, err := json.Marshal(predictableBc)
	c.Assert(err, IsNil)
	c.Check(string(d), Equals, `{"brand-id":"","model":"","grade":"","model-sign-key-id":"","asset-chain":[{"role":"recovery","name":"loader","hashes":["d","e"]},{"role":"recovery","name":"shim","hashes":["b"]},{"role":"run","name":"0oader","hashes":["x","z"]},{"role":"run","name":"1oader","hashes":["d","e"]},{"role":"run","name":"loader","hashes":["z"]},{"role":"run","name":"loader","hashes":["c","d"]}],"kernel":"","kernel-revision":"","kernel-cmdline":""}`)

	// already predictable, but try again
	alreadySortedBc := boot.ToPredictableBootChain(predictableBc)
	c.Check(alreadySortedBc, DeepEquals, predictableBc)

	// boot chain with 2 identical assets
	bcIdenticalAssets := &boot.BootChain{
		AssetChain: []boot.BootAsset{
			{Role: "run", Name: "loader", Hashes: []string{"z"}},
			{Role: "run", Name: "loader", Hashes: []string{"z"}},
		},
	}
	sortedBcIdentical := boot.ToPredictableBootChain(bcIdenticalAssets)
	c.Check(sortedBcIdentical, DeepEquals, bcIdenticalAssets)
}

func (s *bootchainSuite) TestBootChainMarshalFull(c *C) {
	bc := &boot.BootChain{
		BrandID:        "mybrand",
		Model:          "foo",
		Grade:          "dangerous",
		ModelSignKeyID: "my-key-id",
		// asset chain will get sorted when marshaling
		AssetChain: []boot.BootAsset{
			{Role: "run", Name: "loader", Hashes: []string{"c", "d"}},
			// hash list will get sorted
			{Role: "recovery", Name: "shim", Hashes: []string{"b", "a"}},
			{Role: "recovery", Name: "loader", Hashes: []string{"d"}},
		},
		Kernel:         "pc-kernel",
		KernelRevision: "1234",
		KernelCmdline:  `foo=bar baz=0x123`,
	}

	predictableBc := boot.ToPredictableBootChain(bc)

	c.Check(predictableBc, DeepEquals, &boot.BootChain{
		BrandID:        "mybrand",
		Model:          "foo",
		Grade:          "dangerous",
		ModelSignKeyID: "my-key-id",
		// assets are sorted
		AssetChain: []boot.BootAsset{
			{Role: "recovery", Name: "loader", Hashes: []string{"d"}},
			// hash lists are sorted
			{Role: "recovery", Name: "shim", Hashes: []string{"a", "b"}},
			{Role: "run", Name: "loader", Hashes: []string{"c", "d"}},
		},
		Kernel:         "pc-kernel",
		KernelRevision: "1234",
		KernelCmdline:  `foo=bar baz=0x123`,
	})

	d, err := json.Marshal(predictableBc)
	c.Assert(err, IsNil)
	c.Check(string(d), Equals, `{"brand-id":"mybrand","model":"foo","grade":"dangerous","model-sign-key-id":"my-key-id","asset-chain":[{"role":"recovery","name":"loader","hashes":["d"]},{"role":"recovery","name":"shim","hashes":["a","b"]},{"role":"run","name":"loader","hashes":["c","d"]}],"kernel":"pc-kernel","kernel-revision":"1234","kernel-cmdline":"foo=bar baz=0x123"}`)
	// original structure has not been modified
	c.Check(bc, DeepEquals, &boot.BootChain{
		BrandID:        "mybrand",
		Model:          "foo",
		Grade:          "dangerous",
		ModelSignKeyID: "my-key-id",
		// asset chain will get sorted when marshaling
		AssetChain: []boot.BootAsset{
			{Role: "run", Name: "loader", Hashes: []string{"c", "d"}},
			// hash list will get sorted
			{Role: "recovery", Name: "shim", Hashes: []string{"b", "a"}},
			{Role: "recovery", Name: "loader", Hashes: []string{"d"}},
		},
		Kernel:         "pc-kernel",
		KernelRevision: "1234",
		KernelCmdline:  `foo=bar baz=0x123`,
	})
}

func (s *bootchainSuite) TestBootChainEqualForResealComplex(c *C) {
	bc := []boot.BootChain{
		{
			BrandID:        "mybrand",
			Model:          "foo",
			Grade:          "dangerous",
			ModelSignKeyID: "my-key-id",
			AssetChain: []boot.BootAsset{
				{Role: "run", Name: "loader", Hashes: []string{"c", "d"}},
				// hash list will get sorted
				{Role: "recovery", Name: "shim", Hashes: []string{"b", "a"}},
				{Role: "recovery", Name: "loader", Hashes: []string{"d"}},
			},
			Kernel:         "pc-kernel",
			KernelRevision: "1234",
			KernelCmdline:  `foo=bar baz=0x123`,
		},
	}
	pb := boot.ToPredictableBootChains(bc)
	// sorted variant
	pbOther := boot.PredictableBootChains{
		{
			BrandID:        "mybrand",
			Model:          "foo",
			Grade:          "dangerous",
			ModelSignKeyID: "my-key-id",
			AssetChain: []boot.BootAsset{
				{Role: "recovery", Name: "loader", Hashes: []string{"d"}},
				{Role: "recovery", Name: "shim", Hashes: []string{"a", "b"}},
				{Role: "run", Name: "loader", Hashes: []string{"c", "d"}},
			},
			Kernel:         "pc-kernel",
			KernelRevision: "1234",
			KernelCmdline:  `foo=bar baz=0x123`,
		},
	}
	eq := boot.PredictableBootChainsEqualForReseal(pb, pbOther)
	c.Check(eq, Equals, true, Commentf("not equal\none: %v\nother: %v", pb, pbOther))
}

func (s *bootchainSuite) TestPredictableBootChainsEqualForResealSimple(c *C) {
	var pbNil boot.PredictableBootChains

	c.Check(boot.PredictableBootChainsEqualForReseal(pbNil, pbNil), Equals, true)

	bcJustOne := []boot.BootChain{
		{
			BrandID:        "mybrand",
			Model:          "foo",
			Grade:          "dangerous",
			ModelSignKeyID: "my-key-id",
			AssetChain: []boot.BootAsset{
				{Role: "run", Name: "loader", Hashes: []string{"c", "d"}},
			},
			Kernel:         "pc-kernel-other",
			KernelRevision: "1234",
			KernelCmdline:  `foo`,
		},
	}
	pbJustOne := boot.ToPredictableBootChains(bcJustOne)
	// equal with self
	c.Check(boot.PredictableBootChainsEqualForReseal(pbJustOne, pbJustOne), Equals, true)

	// equal with nil?
	c.Check(boot.PredictableBootChainsEqualForReseal(pbJustOne, pbNil), Equals, false)

	bcMoreAssets := []boot.BootChain{
		{
			BrandID:        "mybrand",
			Model:          "foo",
			Grade:          "dangerous",
			ModelSignKeyID: "my-key-id",
			AssetChain: []boot.BootAsset{
				{Role: "run", Name: "loader", Hashes: []string{"c", "d"}},
			},
			Kernel:         "pc-kernel-other",
			KernelRevision: "1234",
			KernelCmdline:  `foo`,
		}, {
			BrandID:        "mybrand",
			Model:          "foo",
			Grade:          "dangerous",
			ModelSignKeyID: "my-key-id",
			AssetChain: []boot.BootAsset{
				{Role: "run", Name: "loader", Hashes: []string{"d", "e"}},
			},
			Kernel:         "pc-kernel-other",
			KernelRevision: "1234",
			KernelCmdline:  `foo`,
		},
	}

	pbMoreAssets := boot.ToPredictableBootChains(bcMoreAssets)

	c.Check(boot.PredictableBootChainsEqualForReseal(pbMoreAssets, pbJustOne), Equals, false)
	// with self
	c.Check(boot.PredictableBootChainsEqualForReseal(pbMoreAssets, pbMoreAssets), Equals, true)
}

func (s *bootchainSuite) TestPredictableBootChainsFullMarshal(c *C) {
	// chains will be sorted
	chains := []boot.BootChain{
		{
			BrandID:        "mybrand",
			Model:          "foo",
			Grade:          "signed",
			ModelSignKeyID: "my-key-id",
			// assets will be sorted
			AssetChain: []boot.BootAsset{
				// hashes will be sorted
				{Role: "recovery", Name: "shim", Hashes: []string{"x", "y"}},
				{Role: "recovery", Name: "loader", Hashes: []string{"c", "d"}},
				{Role: "run", Name: "loader", Hashes: []string{"z", "x"}},
			},
			Kernel:         "pc-kernel-other",
			KernelRevision: "2345",
			KernelCmdline:  `foo`,
		}, {
			BrandID:        "mybrand",
			Model:          "foo",
			Grade:          "dangerous",
			ModelSignKeyID: "my-key-id",
			// assets will be sorted
			AssetChain: []boot.BootAsset{
				// hashes will be sorted
				{Role: "recovery", Name: "shim", Hashes: []string{"y", "x"}},
				{Role: "recovery", Name: "loader", Hashes: []string{"c", "d"}},
				{Role: "run", Name: "loader", Hashes: []string{"b", "a"}},
			},
			Kernel:         "pc-kernel-other",
			KernelRevision: "1234",
			KernelCmdline:  `foo`,
		}, {
			// recovery system
			BrandID:        "mybrand",
			Model:          "foo",
			Grade:          "dangerous",
			ModelSignKeyID: "my-key-id",
			// will be sorted
			AssetChain: []boot.BootAsset{
				{Role: "recovery", Name: "shim", Hashes: []string{"y", "x"}},
				{Role: "recovery", Name: "loader", Hashes: []string{"c", "d"}},
			},
			Kernel:         "pc-kernel-other",
			KernelRevision: "12",
			KernelCmdline:  `foo`,
		},
	}

	predictableChains := boot.ToPredictableBootChains(chains)
	d, err := json.Marshal(predictableChains)
	c.Assert(err, IsNil)

	var data []map[string]interface{}
	err = json.Unmarshal(d, &data)
	c.Assert(err, IsNil)
	c.Check(data, DeepEquals, []map[string]interface{}{
		{
			"model":             "foo",
			"brand-id":          "mybrand",
			"grade":             "dangerous",
			"model-sign-key-id": "my-key-id",
			"kernel":            "pc-kernel-other",
			"kernel-revision":   "12",
			"kernel-cmdline":    "foo",
			"asset-chain": []interface{}{
				map[string]interface{}{"role": "recovery", "name": "loader", "hashes": []interface{}{"c", "d"}},
				map[string]interface{}{"role": "recovery", "name": "shim", "hashes": []interface{}{"x", "y"}},
			},
		}, {
			"model":             "foo",
			"brand-id":          "mybrand",
			"grade":             "dangerous",
			"model-sign-key-id": "my-key-id",
			"kernel":            "pc-kernel-other",
			"kernel-revision":   "1234",
			"kernel-cmdline":    "foo",
			"asset-chain": []interface{}{
				map[string]interface{}{"role": "recovery", "name": "loader", "hashes": []interface{}{"c", "d"}},
				map[string]interface{}{"role": "recovery", "name": "shim", "hashes": []interface{}{"x", "y"}},
				map[string]interface{}{"role": "run", "name": "loader", "hashes": []interface{}{"a", "b"}},
			},
		}, {
			"model":             "foo",
			"brand-id":          "mybrand",
			"grade":             "signed",
			"model-sign-key-id": "my-key-id",
			"kernel":            "pc-kernel-other",
			"kernel-revision":   "2345",
			"kernel-cmdline":    "foo",
			"asset-chain": []interface{}{
				map[string]interface{}{"role": "recovery", "name": "loader", "hashes": []interface{}{"c", "d"}},
				map[string]interface{}{"role": "recovery", "name": "shim", "hashes": []interface{}{"x", "y"}},
				map[string]interface{}{"role": "run", "name": "loader", "hashes": []interface{}{"x", "z"}},
			},
		},
	})
}

func (s *bootchainSuite) TestPredictableBootChainsFields(c *C) {
	chainsNil := boot.ToPredictableBootChains(nil)
	c.Check(chainsNil, IsNil)

	justOne := []boot.BootChain{
		{
			BrandID:        "mybrand",
			Model:          "foo",
			Grade:          "signed",
			ModelSignKeyID: "my-key-id",
			Kernel:         "pc-kernel-other",
			KernelRevision: "2345",
			KernelCmdline:  `foo`,
		},
	}
	predictableJustOne := boot.ToPredictableBootChains(justOne)
	c.Check(predictableJustOne, DeepEquals, boot.PredictableBootChains(justOne))

	chainsGrade := []boot.BootChain{
		{
			Grade: "signed",
		}, {
			Grade: "dangerous",
		},
	}
	c.Check(boot.ToPredictableBootChains(chainsGrade), DeepEquals, boot.PredictableBootChains{
		{
			Grade: "dangerous",
		}, {
			Grade: "signed",
		},
	})

	chainsKernel := []boot.BootChain{
		{
			Grade:  "dangerous",
			Kernel: "foo",
		}, {
			Grade:  "dangerous",
			Kernel: "bar",
		},
	}
	c.Check(boot.ToPredictableBootChains(chainsKernel), DeepEquals, boot.PredictableBootChains{
		{
			Grade:  "dangerous",
			Kernel: "bar",
		}, {
			Grade:  "dangerous",
			Kernel: "foo",
		},
	})

	chainsCmdline := []boot.BootChain{
		{
			Grade:         "dangerous",
			Kernel:        "foo",
			KernelCmdline: `panic=1`,
		}, {
			Grade:         "dangerous",
			Kernel:        "foo",
			KernelCmdline: `a`,
		},
	}
	c.Check(boot.ToPredictableBootChains(chainsCmdline), DeepEquals, boot.PredictableBootChains{
		{
			Grade:         "dangerous",
			Kernel:        "foo",
			KernelCmdline: `a`,
		}, {
			Grade:         "dangerous",
			Kernel:        "foo",
			KernelCmdline: `panic=1`,
		},
	})

	chainsModel := []boot.BootChain{
		{
			Model:         "fridge",
			Grade:         "dangerous",
			Kernel:        "foo",
			KernelCmdline: `panic=1`,
		}, {
			Model:         "box",
			Grade:         "dangerous",
			Kernel:        "foo",
			KernelCmdline: `panic=1`,
		},
	}
	c.Check(boot.ToPredictableBootChains(chainsModel), DeepEquals, boot.PredictableBootChains{
		{
			Model:         "box",
			Grade:         "dangerous",
			Kernel:        "foo",
			KernelCmdline: `panic=1`,
		}, {
			Model:         "fridge",
			Grade:         "dangerous",
			Kernel:        "foo",
			KernelCmdline: `panic=1`,
		},
	})

	chainsBrand := []boot.BootChain{
		{
			BrandID:       "foo",
			Model:         "box",
			Grade:         "dangerous",
			Kernel:        "foo",
			KernelCmdline: `panic=1`,
		}, {
			BrandID:       "acme",
			Model:         "box",
			Grade:         "dangerous",
			Kernel:        "foo",
			KernelCmdline: `panic=1`,
		},
	}
	c.Check(boot.ToPredictableBootChains(chainsBrand), DeepEquals, boot.PredictableBootChains{
		{
			BrandID:       "acme",
			Model:         "box",
			Grade:         "dangerous",
			Kernel:        "foo",
			KernelCmdline: `panic=1`,
		}, {
			BrandID:       "foo",
			Model:         "box",
			Grade:         "dangerous",
			Kernel:        "foo",
			KernelCmdline: `panic=1`,
		},
	})

	chainsKeyID := []boot.BootChain{
		{
			BrandID:        "foo",
			Model:          "box",
			Grade:          "dangerous",
			Kernel:         "foo",
			KernelCmdline:  `panic=1`,
			ModelSignKeyID: "key-2",
		}, {
			BrandID:        "foo",
			Model:          "box",
			Grade:          "dangerous",
			Kernel:         "foo",
			KernelCmdline:  `panic=1`,
			ModelSignKeyID: "key-1",
		},
	}
	c.Check(boot.ToPredictableBootChains(chainsKeyID), DeepEquals, boot.PredictableBootChains{
		{
			BrandID:        "foo",
			Model:          "box",
			Grade:          "dangerous",
			Kernel:         "foo",
			KernelCmdline:  `panic=1`,
			ModelSignKeyID: "key-1",
		}, {
			BrandID:        "foo",
			Model:          "box",
			Grade:          "dangerous",
			Kernel:         "foo",
			KernelCmdline:  `panic=1`,
			ModelSignKeyID: "key-2",
		},
	})

	chainsAssets := []boot.BootChain{
		{
			BrandID:        "foo",
			Model:          "box",
			Grade:          "dangerous",
			ModelSignKeyID: "key-1",
			AssetChain: []boot.BootAsset{
				// will be sorted
				{Hashes: []string{"b", "a"}},
			},
			Kernel:        "foo",
			KernelCmdline: `panic=1`,
		}, {
			BrandID:        "foo",
			Model:          "box",
			Grade:          "dangerous",
			ModelSignKeyID: "key-1",
			AssetChain: []boot.BootAsset{
				{Hashes: []string{"b"}},
			},
			Kernel:        "foo",
			KernelCmdline: `panic=1`,
		},
	}
	c.Check(boot.ToPredictableBootChains(chainsAssets), DeepEquals, boot.PredictableBootChains{
		{
			BrandID:        "foo",
			Model:          "box",
			Grade:          "dangerous",
			ModelSignKeyID: "key-1",
			AssetChain: []boot.BootAsset{
				{Hashes: []string{"b"}},
			},
			Kernel:        "foo",
			KernelCmdline: `panic=1`,
		}, {
			BrandID:        "foo",
			Model:          "box",
			Grade:          "dangerous",
			ModelSignKeyID: "key-1",
			AssetChain: []boot.BootAsset{
				{Hashes: []string{"a", "b"}},
			},
			Kernel:        "foo",
			KernelCmdline: `panic=1`,
		},
	})

	chainsFewerAssets := []boot.BootChain{
		{
			AssetChain: []boot.BootAsset{
				{Hashes: []string{"b", "a"}},
				{Hashes: []string{"c", "d"}},
			},
		}, {
			AssetChain: []boot.BootAsset{
				{Hashes: []string{"b"}},
			},
		},
	}
	c.Check(boot.ToPredictableBootChains(chainsFewerAssets), DeepEquals, boot.PredictableBootChains{
		{
			AssetChain: []boot.BootAsset{
				{Hashes: []string{"b"}},
			},
		}, {
			AssetChain: []boot.BootAsset{
				{Hashes: []string{"a", "b"}},
				{Hashes: []string{"c", "d"}},
			},
		},
	})

	// not confused if 2 chains are identical
	chainsIdenticalAssets := []boot.BootChain{
		{
			BrandID:        "foo",
			Model:          "box",
			ModelSignKeyID: "key-1",
			AssetChain: []boot.BootAsset{
				{Name: "asset", Hashes: []string{"a", "b"}},
				{Name: "asset", Hashes: []string{"a", "b"}},
			},
			Grade:         "dangerous",
			Kernel:        "foo",
			KernelCmdline: `panic=1`,
		}, {
			BrandID:        "foo",
			Model:          "box",
			Grade:          "dangerous",
			ModelSignKeyID: "key-1",
			AssetChain: []boot.BootAsset{
				{Name: "asset", Hashes: []string{"a", "b"}},
				{Name: "asset", Hashes: []string{"a", "b"}},
			},
			Kernel:        "foo",
			KernelCmdline: `panic=1`,
		},
	}
	c.Check(boot.ToPredictableBootChains(chainsIdenticalAssets), DeepEquals, boot.PredictableBootChains(chainsIdenticalAssets))
}

func (s *bootchainSuite) TestPredictableBootChainsSortOrder(c *C) {
	// check that sort order is model info, assets, kernel, kernel cmdline

	chains := []boot.BootChain{
		{
			Model: "b",
			AssetChain: []boot.BootAsset{
				{Name: "asset", Hashes: []string{"y"}},
			},
			Kernel:        "k1",
			KernelCmdline: "cm=1",
		},
		{
			Model: "b",
			AssetChain: []boot.BootAsset{
				{Name: "asset", Hashes: []string{"y"}},
			},
			Kernel:        "k2",
			KernelCmdline: "cm=1",
		},
		{
			Model: "a",
			AssetChain: []boot.BootAsset{
				{Name: "asset", Hashes: []string{"y"}},
			},
			Kernel:        "k1",
			KernelCmdline: "cm=1",
		},
		{
			Model: "a",
			AssetChain: []boot.BootAsset{
				{Name: "asset", Hashes: []string{"y"}},
			},
			Kernel:        "k2",
			KernelCmdline: "cm=1",
		},
		{
			Model: "b",
			AssetChain: []boot.BootAsset{
				{Name: "asset", Hashes: []string{"y"}},
			},
			Kernel:        "k1",
			KernelCmdline: "cm=2",
		},
		{
			Model: "b",
			AssetChain: []boot.BootAsset{
				{Name: "asset", Hashes: []string{"y"}},
			},
			Kernel:        "k2",
			KernelCmdline: "cm=2",
		},
		{
			Model: "a",
			AssetChain: []boot.BootAsset{
				{Name: "asset", Hashes: []string{"y"}},
			},
			Kernel:        "k1",
			KernelCmdline: "cm=2",
		},
		{
			Model: "a",
			AssetChain: []boot.BootAsset{
				{Name: "asset", Hashes: []string{"y"}},
			},
			Kernel:        "k2",
			KernelCmdline: "cm=2",
		},
		{
			Model: "b",
			AssetChain: []boot.BootAsset{
				{Name: "asset", Hashes: []string{"x"}},
			},
			Kernel:        "k1",
			KernelCmdline: "cm=1",
		},
		{
			Model: "b",
			AssetChain: []boot.BootAsset{
				{Name: "asset", Hashes: []string{"x"}},
			},
			Kernel:        "k2",
			KernelCmdline: "cm=1",
		},
		{
			Model: "a",
			AssetChain: []boot.BootAsset{
				{Name: "asset", Hashes: []string{"x"}},
			},
			Kernel:        "k1",
			KernelCmdline: "cm=1",
		},
		{
			Model: "a",
			AssetChain: []boot.BootAsset{
				{Name: "asset", Hashes: []string{"x"}},
			},
			Kernel:        "k2",
			KernelCmdline: "cm=1",
		},
		{
			Model: "b",
			AssetChain: []boot.BootAsset{
				{Name: "asset", Hashes: []string{"x"}},
			},
			Kernel:        "k1",
			KernelCmdline: "cm=2",
		},
		{
			Model: "b",
			AssetChain: []boot.BootAsset{
				{Name: "asset", Hashes: []string{"x"}},
			},
			Kernel:        "k2",
			KernelCmdline: "cm=2",
		},
		{
			Model: "a",
			AssetChain: []boot.BootAsset{
				{Name: "asset", Hashes: []string{"x"}},
			},
			Kernel:        "k1",
			KernelCmdline: "cm=2",
		},
		{
			Model: "a",
			AssetChain: []boot.BootAsset{
				{Name: "asset", Hashes: []string{"x"}},
			},
			Kernel:        "k2",
			KernelCmdline: "cm=2",
		},
	}
	predictable := boot.ToPredictableBootChains(chains)
	c.Check(predictable, DeepEquals, boot.PredictableBootChains{
		{
			Model: "a",
			AssetChain: []boot.BootAsset{
				{Name: "asset", Hashes: []string{"x"}},
			},
			Kernel:        "k1",
			KernelCmdline: "cm=1",
		},
		{
			Model: "a",
			AssetChain: []boot.BootAsset{
				{Name: "asset", Hashes: []string{"x"}},
			},
			Kernel:        "k1",
			KernelCmdline: "cm=2",
		},
		{
			Model: "a",
			AssetChain: []boot.BootAsset{
				{Name: "asset", Hashes: []string{"x"}},
			},
			Kernel:        "k2",
			KernelCmdline: "cm=1",
		},
		{
			Model: "a",
			AssetChain: []boot.BootAsset{
				{Name: "asset", Hashes: []string{"x"}},
			},
			Kernel:        "k2",
			KernelCmdline: "cm=2",
		},
		{
			Model: "a",
			AssetChain: []boot.BootAsset{
				{Name: "asset", Hashes: []string{"y"}},
			},
			Kernel:        "k1",
			KernelCmdline: "cm=1",
		},
		{
			Model: "a",
			AssetChain: []boot.BootAsset{
				{Name: "asset", Hashes: []string{"y"}},
			},
			Kernel:        "k1",
			KernelCmdline: "cm=2",
		},
		{
			Model: "a",
			AssetChain: []boot.BootAsset{
				{Name: "asset", Hashes: []string{"y"}},
			},
			Kernel:        "k2",
			KernelCmdline: "cm=1",
		},
		{
			Model: "a",
			AssetChain: []boot.BootAsset{
				{Name: "asset", Hashes: []string{"y"}},
			},
			Kernel:        "k2",
			KernelCmdline: "cm=2",
		},
		{
			Model: "b",
			AssetChain: []boot.BootAsset{
				{Name: "asset", Hashes: []string{"x"}},
			},
			Kernel:        "k1",
			KernelCmdline: "cm=1",
		},
		{
			Model: "b",
			AssetChain: []boot.BootAsset{
				{Name: "asset", Hashes: []string{"x"}},
			},
			Kernel:        "k1",
			KernelCmdline: "cm=2",
		},
		{
			Model: "b",
			AssetChain: []boot.BootAsset{
				{Name: "asset", Hashes: []string{"x"}},
			},
			Kernel:        "k2",
			KernelCmdline: "cm=1",
		},
		{
			Model: "b",
			AssetChain: []boot.BootAsset{
				{Name: "asset", Hashes: []string{"x"}},
			},
			Kernel:        "k2",
			KernelCmdline: "cm=2",
		},
		{
			Model: "b",
			AssetChain: []boot.BootAsset{
				{Name: "asset", Hashes: []string{"y"}},
			},
			Kernel:        "k1",
			KernelCmdline: "cm=1",
		},
		{
			Model: "b",
			AssetChain: []boot.BootAsset{
				{Name: "asset", Hashes: []string{"y"}},
			},
			Kernel:        "k1",
			KernelCmdline: "cm=2",
		},
		{
			Model: "b",
			AssetChain: []boot.BootAsset{
				{Name: "asset", Hashes: []string{"y"}},
			},
			Kernel:        "k2",
			KernelCmdline: "cm=1",
		},
		{
			Model: "b",
			AssetChain: []boot.BootAsset{
				{Name: "asset", Hashes: []string{"y"}},
			},
			Kernel:        "k2",
			KernelCmdline: "cm=2",
		},
	})
}

func printChain(c *C, chain *secboot.LoadChain, prefix string) {
	c.Logf("%v %v", prefix, chain.BootFile)
	for _, n := range chain.Next {
		printChain(c, n, prefix+"-")
	}
}

// cPath returns a path under boot assets cache directory
func cPath(p string) string {
	return filepath.Join(dirs.SnapBootAssetsDir, p)
}

// nbf is bootloader.NewBootFile but shorter
func nbf(snap, path, role string) bootloader.BootFile {
	return bootloader.NewBootFile(snap, path, bootloader.Role(role))
}

func (s *bootchainSuite) TestBootAssetsToLoadChainTrivialKernel(c *C) {
	kbl := bootloader.NewBootFile("pc-kernel", "kernel.efi", bootloader.Role("run"))

	chains, err := boot.BootAssetsToLoadChains(nil, kbl, nil)
	c.Assert(err, IsNil)

	c.Check(chains, DeepEquals, []*secboot.LoadChain{
		secboot.NewLoadChain(nbf("pc-kernel", "kernel.efi", "run")),
	})
}

func (s *bootchainSuite) TestBootAssetsToLoadChainErr(c *C) {
	kbl := bootloader.NewBootFile("pc-kernel", "kernel.efi", bootloader.Role("run"))

	assets := []boot.BootAsset{
		{Name: "shim", Hashes: []string{"hash0"}, Role: "recovery"},
		{Name: "loader-recovery", Hashes: []string{"hash0"}, Role: "recovery"},
		{Name: "loader-run", Hashes: []string{"hash0"}, Role: "run"},
	}

	blNames := map[bootloader.Role]string{
		"recovery": "recovery-bl",
		// missing bootloader name for role "run"
	}
	// fails when probing the shim asset in the cache
	chains, err := boot.BootAssetsToLoadChains(assets, kbl, blNames)
	c.Assert(err, ErrorMatches, "file .*/recovery-bl/shim-hash0 not found in boot assets cache")
	c.Check(chains, IsNil)
	// make it work now
	c.Assert(os.MkdirAll(filepath.Dir(cPath("recovery-bl/shim-hash0")), 0755), IsNil)
	c.Assert(ioutil.WriteFile(cPath("recovery-bl/shim-hash0"), nil, 0644), IsNil)

	// nested error bubbled up
	chains, err = boot.BootAssetsToLoadChains(assets, kbl, blNames)
	c.Assert(err, ErrorMatches, "file .*/recovery-bl/loader-recovery-hash0 not found in boot assets cache")
	c.Check(chains, IsNil)
	// again, make it work
	c.Assert(os.MkdirAll(filepath.Dir(cPath("recovery-bl/loader-recovery-hash0")), 0755), IsNil)
	c.Assert(ioutil.WriteFile(cPath("recovery-bl/loader-recovery-hash0"), nil, 0644), IsNil)

	// fails on missing bootloader name for role "run"
	chains, err = boot.BootAssetsToLoadChains(assets, kbl, blNames)
	c.Assert(err, ErrorMatches, `internal error: no bootloader name for boot asset role "run"`)
	c.Check(chains, IsNil)
}

func (s *bootchainSuite) TestBootAssetsToLoadChainSimpleChain(c *C) {
	kbl := bootloader.NewBootFile("pc-kernel", "kernel.efi", bootloader.Role("run"))

	assets := []boot.BootAsset{
		{Name: "shim", Hashes: []string{"hash0"}, Role: "recovery"},
		{Name: "loader-recovery", Hashes: []string{"hash0"}, Role: "recovery"},
		{Name: "loader-run", Hashes: []string{"hash0"}, Role: "run"},
	}

	// mock relevant files in cache
	for _, name := range []string{
		"recovery-bl/shim-hash0",
		"recovery-bl/loader-recovery-hash0",
		"run-bl/loader-run-hash0",
	} {
		p := filepath.Join(dirs.SnapBootAssetsDir, name)
		c.Assert(os.MkdirAll(filepath.Dir(p), 0755), IsNil)
		c.Assert(ioutil.WriteFile(p, nil, 0644), IsNil)
	}

	blNames := map[bootloader.Role]string{
		"recovery": "recovery-bl",
		"run":      "run-bl",
	}

	chains, err := boot.BootAssetsToLoadChains(assets, kbl, blNames)
	c.Assert(err, IsNil)

	c.Logf("got:")
	for _, ch := range chains {
		printChain(c, ch, "-")
	}

	expected := []*secboot.LoadChain{
		secboot.NewLoadChain(nbf("", cPath("recovery-bl/shim-hash0"), "recovery"),
			secboot.NewLoadChain(nbf("", cPath("recovery-bl/loader-recovery-hash0"), "recovery"),
				secboot.NewLoadChain(nbf("", cPath("run-bl/loader-run-hash0"), "run"),
					secboot.NewLoadChain(nbf("pc-kernel", "kernel.efi", "run"))))),
	}
	c.Check(chains, DeepEquals, expected)
}

func (s *bootchainSuite) TestBootAssetsToLoadChainWithAlternativeChains(c *C) {
	kbl := bootloader.NewBootFile("pc-kernel", "kernel.efi", bootloader.Role("run"))

	assets := []boot.BootAsset{
		{Name: "shim", Hashes: []string{"hash0", "hash1"}, Role: "recovery"},
		{Name: "loader-recovery", Hashes: []string{"hash0", "hash1"}, Role: "recovery"},
		{Name: "loader-run", Hashes: []string{"hash0", "hash1"}, Role: "run"},
	}

	// mock relevant files in cache
	for _, name := range []string{
		"recovery-bl/shim-hash0", "recovery-bl/shim-hash1",
		"recovery-bl/loader-recovery-hash0",
		"recovery-bl/loader-recovery-hash1",
		"run-bl/loader-run-hash0",
		"run-bl/loader-run-hash1",
	} {
		p := filepath.Join(dirs.SnapBootAssetsDir, name)
		c.Assert(os.MkdirAll(filepath.Dir(p), 0755), IsNil)
		c.Assert(ioutil.WriteFile(p, nil, 0644), IsNil)
	}

	blNames := map[bootloader.Role]string{
		"recovery": "recovery-bl",
		"run":      "run-bl",
	}
	chains, err := boot.BootAssetsToLoadChains(assets, kbl, blNames)
	c.Assert(err, IsNil)

	c.Logf("got:")
	for _, ch := range chains {
		printChain(c, ch, "-")
	}

	expected := []*secboot.LoadChain{
		secboot.NewLoadChain(nbf("", cPath("recovery-bl/shim-hash0"), "recovery"),
			secboot.NewLoadChain(nbf("", cPath("recovery-bl/loader-recovery-hash0"), "recovery"),
				secboot.NewLoadChain(nbf("", cPath("run-bl/loader-run-hash0"), "run"),
					secboot.NewLoadChain(nbf("pc-kernel", "kernel.efi", "run"))),
				secboot.NewLoadChain(nbf("", cPath("run-bl/loader-run-hash1"), "run"),
					secboot.NewLoadChain(nbf("pc-kernel", "kernel.efi", "run")))),
			secboot.NewLoadChain(nbf("", cPath("recovery-bl/loader-recovery-hash1"), "recovery"),
				secboot.NewLoadChain(nbf("", cPath("run-bl/loader-run-hash0"), "run"),
					secboot.NewLoadChain(nbf("pc-kernel", "kernel.efi", "run"))),
				secboot.NewLoadChain(nbf("", cPath("run-bl/loader-run-hash1"), "run"),
					secboot.NewLoadChain(nbf("pc-kernel", "kernel.efi", "run"))))),
		secboot.NewLoadChain(nbf("", cPath("recovery-bl/shim-hash1"), "recovery"),
			secboot.NewLoadChain(nbf("", cPath("recovery-bl/loader-recovery-hash0"), "recovery"),
				secboot.NewLoadChain(nbf("", cPath("run-bl/loader-run-hash0"), "run"),
					secboot.NewLoadChain(nbf("pc-kernel", "kernel.efi", "run"))),
				secboot.NewLoadChain(nbf("", cPath("run-bl/loader-run-hash1"), "run"),
					secboot.NewLoadChain(nbf("pc-kernel", "kernel.efi", "run")))),
			secboot.NewLoadChain(nbf("", cPath("recovery-bl/loader-recovery-hash1"), "recovery"),
				secboot.NewLoadChain(nbf("", cPath("run-bl/loader-run-hash0"), "run"),
					secboot.NewLoadChain(nbf("pc-kernel", "kernel.efi", "run"))),
				secboot.NewLoadChain(nbf("", cPath("run-bl/loader-run-hash1"), "run"),
					secboot.NewLoadChain(nbf("pc-kernel", "kernel.efi", "run"))))),
	}
	c.Check(chains, DeepEquals, expected)
}
