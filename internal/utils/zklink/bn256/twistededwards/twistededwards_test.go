/*
Copyright © 2020 ConsenSys

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package twistededwards

import (
	"testing"

	"github.com/thrasher-corp/gocryptotrader/internal/utils/zklink/bn256/fr"
)

func TestAdd(t *testing.T) {

	var p1, p2 Point

	p1.X.SetString("8728367628344135467582547753719073727968275979035063555332785894244029982715")
	p1.Y.SetString("8834462946188529904793384347374734779374831553974460136522409595751449858199")

	p2.X.SetString("9560056125663567360314373555170485462871740364163814576088225107862234393497")
	p2.Y.SetString("13024071698463677601393829581435828705327146000694268918451707151508990195684")

	var expectedX, expectedY fr.Element

	expectedX.SetString("15602730788680306249246507407102613100672389871136626657306339018592280799798")
	expectedY.SetString("9214827499166027327226786359816287546740571844393227610238633031200971415079")

	p1.Add(&p1, &p2)

	if !p1.X.Equal(&expectedX) {
		t.Fatal("wrong x coordinate")
	}
	if !p1.Y.Equal(&expectedY) {
		t.Fatal("wrong y coordinate")
	}
}

func TestAddProj(t *testing.T) {
	var p1, p2 Point
	var p1proj, p2proj PointProj

	p1.X.SetString("8728367628344135467582547753719073727968275979035063555332785894244029982715")
	p1.Y.SetString("8834462946188529904793384347374734779374831553974460136522409595751449858199")

	p2.X.SetString("9560056125663567360314373555170485462871740364163814576088225107862234393497")
	p2.Y.SetString("13024071698463677601393829581435828705327146000694268918451707151508990195684")

	p1proj.FromAffine(&p1)
	p2proj.FromAffine(&p2)

	var expectedX, expectedY fr.Element

	expectedX.SetString("15602730788680306249246507407102613100672389871136626657306339018592280799798")
	expectedY.SetString("9214827499166027327226786359816287546740571844393227610238633031200971415079")

	p1proj.Add(&p1proj, &p2proj)
	p1.FromProj(&p1proj)

	if !p1.X.Equal(&expectedX) {
		t.Fatal("wrong x coordinate")
	}
	if !p1.Y.Equal(&expectedY) {
		t.Fatal("wrong y coordinate")
	}
}

func TestDouble(t *testing.T) {
	var p Point
	p.X.SetString("8728367628344135467582547753719073727968275979035063555332785894244029982715")
	p.Y.SetString("8834462946188529904793384347374734779374831553974460136522409595751449858199")

	p.Double(&p)

	var expectedX, expectedY fr.Element

	expectedX.SetString("17048188201798084482613703497237052386773720266456818725024051932759787099830")
	expectedY.SetString("15722506141850766164380928609287974914029282300941585435780118880890915697552")

	if !p.X.Equal(&expectedX) {
		t.Fatal("wrong x coordinate")
	}
	if !p.Y.Equal(&expectedY) {
		t.Fatal("wrong y coordinate")
	}
}

func TestDoubleProj(t *testing.T) {

	var p Point
	var pproj PointProj

	p.X.SetString("8728367628344135467582547753719073727968275979035063555332785894244029982715")
	p.Y.SetString("8834462946188529904793384347374734779374831553974460136522409595751449858199")

	pproj.FromAffine(&p).Double(&pproj)

	p.FromProj(&pproj)

	var expectedX, expectedY fr.Element

	expectedX.SetString("17048188201798084482613703497237052386773720266456818725024051932759787099830")
	expectedY.SetString("15722506141850766164380928609287974914029282300941585435780118880890915697552")

	if !p.X.Equal(&expectedX) {
		t.Fatal("wrong x coordinate")
	}
	if !p.Y.Equal(&expectedY) {
		t.Fatal("wrong y coordinate")
	}
}

func TestScalarMul(t *testing.T) {

	// set curve parameters
	ed := GetEdwardsCurve()

	var scalar fr.Element
	scalar.SetUint64(23902374).FromMont()

	var p Point
	p.ScalarMul(&ed.Base, scalar)

	var expectedX, expectedY fr.Element

	expectedX.SetString("2617519824163134005353570974989848134508856877236793995668417237392062754831")
	expectedY.SetString("12956808000482532416873382696451950668786244907047953547021024966691314258300")

	if !expectedX.Equal(&p.X) {
		t.Fatal("wrong x coordinate")
	}
	if !expectedY.Equal(&p.Y) {
		t.Fatal("wrong y coordinate")
	}
}
