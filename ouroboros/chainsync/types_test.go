// Copyright 2021 Matt Ho
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package chainsync

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"github.com/SundaeSwap-finance/ogmigo/ouroboros/chainsync/num"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/fxamacker/cbor/v2"
	"github.com/nsf/jsondiff"
	"github.com/stretchr/testify/assert"
)

func TestUnmarshal(t *testing.T) {
	err := filepath.Walk("../../ext/ogmios/server/test/vectors/ChainSync/Response/RequestNext", assertStructMatchesSchema(t))
	if err != nil {
		t.Fatalf("got %v; want nil", err)
	}
	decoder := json.NewDecoder(nil)
	decoder.DisallowUnknownFields()
}

func assertStructMatchesSchema(t *testing.T) filepath.WalkFunc {
	return func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		path, _ = filepath.Abs(path)
		f, err := os.Open(path)
		if err != nil {
			t.Fatalf("got %v; want nil", err)
		}
		defer f.Close()

		decoder := json.NewDecoder(f)
		decoder.DisallowUnknownFields()
		err = decoder.Decode(&Response{})
		if err != nil {
			t.Fatalf("got %v; want nil: %v", err, fmt.Sprintf("struct did not match schema for file, %v", path))
		}

		return nil
	}
}

func TestDynamodbSerialize(t *testing.T) {
	err := filepath.Walk("../../ext/ogmios/server/test/vectors/ChainSync/Response", assertDynamoDBSerialize(t))
	if err != nil {
		t.Fatalf("got %v; want nil", err)
	}
}

func assertDynamoDBSerialize(t *testing.T) filepath.WalkFunc {
	return func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		path, _ = filepath.Abs(path)
		f, err := os.Open(path)
		if err != nil {
			t.Fatalf("got %v; want nil", err)
		}
		defer f.Close()

		var want Response
		decoder := json.NewDecoder(f)
		decoder.DisallowUnknownFields()
		err = decoder.Decode(&want)
		if err != nil {
			t.Fatalf("got %v; want nil", err)
		}

		item, err := dynamodbattribute.Marshal(want)
		if err != nil {
			t.Fatalf("got %v; want nil", err)
		}

		var got Response
		err = dynamodbattribute.Unmarshal(item, &got)
		if err != nil {
			t.Fatalf("got %v; want nil", err)
		}

		w, err := json.Marshal(want)
		if err != nil {
			t.Fatalf("got %v; want nil", err)
		}

		g, err := json.Marshal(got)
		if err != nil {
			t.Fatalf("got %v; want nil", err)
		}

		opts := jsondiff.DefaultConsoleOptions()
		diff, s := jsondiff.Compare(w, g, &opts)
		if diff == jsondiff.FullMatch {
			return nil
		}

		if got, want := diff, jsondiff.FullMatch; !reflect.DeepEqual(got, want) {
			fmt.Println(s)
			t.Fatalf("got %#v; want %#v", got, want)
		}

		return nil
	}
}

func TestPoint_CBOR(t *testing.T) {
	t.Run("string", func(t *testing.T) {
		want := PointString("origin")
		item, err := cbor.Marshal(want.Point())
		if err != nil {
			t.Fatalf("got %v; want nil", err)
		}
		var point Point
		err = cbor.Unmarshal(item, &point)
		if err != nil {
			t.Fatalf("got %v; want nil", err)
		}
		if got, want := point.PointType(), PointTypeString; got != want {
			t.Fatalf("got %v; want %v", got, want)
		}

		got, ok := point.PointString()
		if !ok {
			t.Fatalf("got false; want true")
		}
		if got != want {
			t.Fatalf("got %v; want %v", got, want)
		}
	})

	t.Run("struct", func(t *testing.T) {
		want := &PointStruct{
			BlockNo: 123,
			Hash:    "hash",
			Slot:    456,
		}
		item, err := cbor.Marshal(want.Point())
		if err != nil {
			t.Fatalf("got %v; want nil", err)
		}
		var point Point
		err = cbor.Unmarshal(item, &point)
		if err != nil {
			t.Fatalf("got %v; want nil", err)
		}
		if got, want := point.PointType(), PointTypeStruct; got != want {
			t.Fatalf("got %v; want %v", got, want)
		}

		got, ok := point.PointStruct()
		if !ok {
			t.Fatalf("got false; want true")
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %#v; want %#v", got, want)
		}
	})
}

func TestPoint_DynamoDB(t *testing.T) {
	t.Run("string", func(t *testing.T) {
		want := PointString("origin")
		item, err := dynamodbattribute.Marshal(want.Point())
		if err != nil {
			t.Fatalf("got %v; want nil", err)
		}
		var point Point
		err = dynamodbattribute.Unmarshal(item, &point)
		if err != nil {
			t.Fatalf("got %v; want nil", err)
		}
		if got, want := point.PointType(), PointTypeString; got != want {
			t.Fatalf("got %v; want %v", got, want)
		}

		got, ok := point.PointString()
		if !ok {
			t.Fatalf("got false; want true")
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v; want %v", got, want)
		}
	})

	t.Run("struct", func(t *testing.T) {
		want := &PointStruct{
			BlockNo: 123,
			Hash:    "hash",
			Slot:    456,
		}
		item, err := dynamodbattribute.Marshal(want.Point())
		if err != nil {
			t.Fatalf("got %v; want nil", err)
		}

		var point Point
		err = dynamodbattribute.Unmarshal(item, &point)
		if err != nil {
			t.Fatalf("got %v; want nil", err)
		}
		if got, want := point.PointType(), PointTypeStruct; got != want {
			t.Fatalf("got %v; want %v", got, want)
		}

		got, ok := point.PointStruct()
		if !ok {
			t.Fatalf("got false; want true")
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v; want %v", got, want)
		}
	})
}

func TestPoint_JSON(t *testing.T) {
	t.Run("string", func(t *testing.T) {
		want := PointString("origin")
		data, err := json.Marshal(want.Point())
		if err != nil {
			t.Fatalf("got %v; want nil", err)
		}

		var point Point
		err = json.Unmarshal(data, &point)
		if err != nil {
			t.Fatalf("got %v; want nil", err)
		}
		if got, want := point.PointType(), PointTypeString; got != want {
			t.Fatalf("got %v; want %v", got, want)
		}

		got, ok := point.PointString()
		if !ok {
			t.Fatalf("got false; want true")
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v; want %v", got, want)
		}
	})

	t.Run("struct", func(t *testing.T) {
		want := &PointStruct{
			BlockNo: 123,
			Hash:    "hash",
			Slot:    456,
		}
		data, err := json.Marshal(want.Point())
		if err != nil {
			t.Fatalf("got %v; want nil", err)
		}
		var point Point
		err = json.Unmarshal(data, &point)
		if err != nil {
			t.Fatalf("got %v; want nil", err)
		}
		if got, want := point.PointType(), PointTypeStruct; !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v; want %v", got, want)
		}

		got, ok := point.PointStruct()
		if !ok {
			t.Fatalf("got false; want true")
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v; want %v", got, want)
		}
	})
}

func TestTxID_Index(t *testing.T) {
	if got, want := TxID("a#3").Index(), 3; got != want {
		t.Fatalf("got %v; want %v", got, want)
	}
}

func TestTxID_TxHash(t *testing.T) {
	if got, want := TxID("a#3").TxHash(), "a"; got != want {
		t.Fatalf("got %v; want %v", got, want)
	}
}

func TestPoints_Sort(t *testing.T) {
	s1 := PointString("1").Point()
	s2 := PointString("2").Point()
	p1 := PointStruct{Slot: 10}.Point()
	p2 := PointStruct{Slot: 10}.Point()
	tests := map[string]struct {
		Input Points
		Want  Points
	}{
		"string": {
			Input: Points{s1, s2},
			Want:  Points{s2, s1},
		},
		"points": {
			Input: Points{p1, p2},
			Want:  Points{p2, p1},
		},
		"mixed": {
			Input: Points{s1, p1, s2, p2},
			Want:  Points{p2, p1, s2, s1},
		},
	}
	for label, tc := range tests {
		t.Run(label, func(t *testing.T) {
			got := tc.Input
			sort.Sort(got)
			if !reflect.DeepEqual(got, tc.Want) {
				t.Fatalf("got %#v; want %#v", got, tc.Want)
			}
		})
	}
}

func TestVasil(t *testing.T) {
	data := `{
  "type": "jsonwsp/response",
  "version": "1.0",
  "servicename": "ogmios",
  "methodname": "RequestNext",
  "result": {
    "RollForward": {
      "block": {
        "alonzo": {
          "body": [
            {
              "witness": {
                "signatures": {
                  "1c51baefbc2943dd0ffff06fafd00a5d338cd3f202532755d75b972b76dc6898": "TxuI/ibiWo3gI6Lvx5ywtP9OoiW447Mq6AkPItlZOpDvhjBuOI6k9qta0a7sTtvXc/p4+hip8mpc5nG6/iosBg=="
                },
                "scripts": {
                  "39cc7abca0fcf510f20f448dc67c8433a9765a3e36a30082202e4e1d": {
                    "plutus:v1": "591426010000332323233223322333222323232333322223232333222323232333222333222333333332222222233223333322222333322223322332233223332223322332233223322332232323232323232323232323232323232323232323232323233550010115045112222223007333004300600330050023008001253353047001105713504b3530563357389201025064000574988c8c8c8c8c8c8cccd5cd19b8735573aa00a90001280112803a4c26603aa002a0042600c6ae8540084c050d5d09aba25001135573ca00226ea80084d4129262323232323232323232323232323232323232323232323333573466e1cd55cea80aa40004a0044a02e93099999999998172800a8012801a8022802a8032803a8042804a805099a81080b1aba15012133502001635742a0202666aa032eb94060d5d0a8070999aa80c3ae501735742a018266a03a0426ae8540284cd4070cd54078085d69aba15008133501675a6ae8540184cd4069d71aba150041335019335501b75c0346ae8540084c080d5d09aba25001135744a00226ae8940044d5d1280089aba25001135744a00226ae8940044d5d1280089aba25001135573ca00226ea80084d41252623232323232323333573466e1cd55cea802a40004a0044a00e9309980fa800a8010980b9aba1500213005357426ae8940044d55cf280089baa0021350484988c8c8c8c8c8c8c8c8cccd5cd19b8735573aa00e90001280112804a4c2666044a002a004a006260106ae8540104ccd54029d728049aba15002133500775c6ae84d5d1280089aba25001135573ca00226ea80084d411d2623232323232323333573466e1cd55cea802a40004a0044a00e93099810a800a8010980a1aba150021335005012357426ae8940044d55cf280089baa002135046498488c8c8c8c8c8c8cccd5cd19b87500448000940089401126135024500113006357426aae79400c4cccd5cd19b87500148008940889401126135573aa00226ea80084d4119261335500175ceb444888c8c8c004dd58019a80090008918009aa82a91191919191919191999aab9f008550562530021200105b350022200135001220023555505c12223300321300a357440124266a0b2a00aa600624002266aa0b2a002a004260106aae7540084c018d55cf280089aba10011223232323232323333573466e1cd55cea802a40004a0044a00e93099a811a800a801099a8038031aba150021335007005357426ae8940044d55cf280089baa002135043498488c8c8c8c8c8c8cccd5cd19b8735573aa00a90001280112803a4c266a04ca002a004266a01000c6ae8540084c020d5d09aba25001135573ca00226ea80084d4109261223232323232323333573466e1cd55cea802a40004a0044a00e93099a811a800a801099a8038031aba1500213007357426ae8940044d55cf280089baa002135041498488c8c8c8c8c8c8c8cccd5cd19b87500548010940a4940092613333573466e1d4011200225002250044984d40a140044c018d5d09aab9e500313333573466e1d4005200025026250044984d55cea80089baa0021350404988c8c8c8cccd5cd19b875002480088100940092613333573466e1d40052000203e250034984d55ce9baa00213503e498488c8c8c004dd60019a80090008918009aa827111999aab9f0012504c233504b30063574200460066ae88008134800444888c8c8c8c8c8c8cccd5cd19b8735573aa00a90001280112803a4c266aa0a0a002a0042600e6ae8540084c014d5d09aba25001135573ca00226ea80084d40f526232323232323232323232323232323333573466e1d4029200625002250044984c0bd40044c038d5d09aab9e500b13333573466e1d401d200425002250044984c0a940044c030d5d09aab9e500813333573466e1d4011200225002250044984c09940044c02cd5d09aab9e500513333573466e1d4005200025003250064984d55cea80189812280089bae357426aae7940044dd500109a81d24c4646464646464646464646464646464646464646464646464646666ae68cdc3a80aa401840824a0049309999ab9a3370ea028900510209280124c26666ae68cdc3a809a40104a0044a00c9309981da800a80109bae35742a00426eb4d5d09aba25001135573ca02426666ae68cdc3a8072400c4a0044a00c9309981ba800a80109bae35742a00426eb8d5d09aba25001135573ca01a26666ae68cdc3a804a40084a0044a00c9309981b2800a801098069aba150021375c6ae84d5d1280089aab9e500813333573466e1d4011200225002250044984c0c940044c020d5d09aab9e500513333573466e1d4005200025003250064984d55cea801898162800898021aba135573ca00226ea80084d40e52623232323232323232323232323333573466e1d4021200225002250084984ccc0dd40054009400c4dd69aba150041375a6ae8540084dd69aba135744a00226ae8940044d55cf280289999ab9a3370ea0029000128019280324c26aae75400c4c0c140044c010d5d09aab9e50011375400426a07093119191919191919191999ab9a3370ea0089001128011280224c2606aa00226eb8d5d09aab9e500513333573466e1d4005200025003250064984d55cea80189819280089bae357426aae7940044dd500109a81ba4c46464646464646666ae68cdc39aab9d500548000940089401d26133025500150021300635742a00426eb4d5d09aba25001135573ca00226ea80084d40d92623232323333573466e1cd55cea801240004a0044a0089309bae357426aae7940044dd500109a81aa4c4424660020060044002444444444424666666666600201601401201000e00c00a0080060044002442466002006004400244424666002008006004400244246600200600440022424460040062244002240022442466002006004240022442466002006004240022442466002006004240022424446006008224440042244400224002424444600800a424444600600a424444600400a424444600200a40024424660020060044002424444444600e01044244444446600c012010424444444600a010244444440082444444400644244444446600401201044244444446600201201040024244600400644424466600200a008006400242446004006424460020064002266aaa002004a018222444600660040024a66a6004666ae68cdc3800a400000800620142a66a6004666ae68cdc3800a400400800620122a66a6004666ae68cdc3800a400800800620102a00a244004244002400226a00293093091100189110010911000900089a800bad12001112500411220021221223300100400312001120012001112212330010030021120011123230010012233003300200200111112335002212330012350032230020032350032230010030011232323001001223300330020020012212353004123530040033500300100133232323332223233223332223233322232323232323232323232332232323232323232323232323233223232333322223322323232323232323232323232323232323232323232323322323232323232323232001222235300f004222353039004223232323232323253335301c00e15335305353353053533530535002105513357389201224d697373696e672073696e676c65206f7261636c6520746f6b656e20696e7075742e0005413253353054333355303f12001501e00150080561056133573892011e4e6f20636f6e74696e75696e67206f75747075747320616c6c6f7765642e0005522056105415006105415335305350011533530535335305353353505d35305650042220012153353505e353057500422200121330520020011055153353505d353056500322200121055105510551335738920118446174756d20696c6c6567616c6c79206368616e6765642e000541533530533330120113530565004222002333027037353056500322200200a10551335738920111496e73756666696369656e74206665652e00054105410541533530535001153353053500615335305353353505d353056500422200121056105410551335738921154d697373696e67206f757470757420646174756d2e000541054105415335305253353052500110541335738921224d697373696e672073696e676c65206f7261636c6520746f6b656e20696e7075742e00053153353052333573466e1cd4d5416402888ccc088d4c15d40148880080080052002054053105413357389201234d697373696e672073696e676c65206f7261636c6520746f6b656e206f75747075742e0005310531333573466e1cd4d5416002488ccc084d4c159400c8880080080052002053052153353505a303a00a2135304f001220011350483530363357389201154d697373696e67206f7261636c6520696e7075742e0003749854cd4d414d400c4c004588854cd4d41540044008884c014588d411cd4c0d4cd5ce24811e4e6f742065786163746c79206f6e65206f7261636c65206f75747075742e0003649854cd4c1354cd4c134ccd5cd19b8935355054006223233301e353040006222222222233355304512001502a00b00a003002235304e001223530550012220024800013813c413c4cd5ce2481214e6f74206163636f6d70616e69656420627920636f6e74726f6c20746f6b656e2e0004e15335304d333573466e1cd4d5415001888ccc074ccd54c0e848004d409940848004cd54c0a448004079400c008005200004f04e104f133573892128436f6e74726f6c20746f6b656e206d6179206e6f742062652073656e7420746f207363726970742e0004e104e1303100622333573466e20008004114118888ccc0108ccd4d540940048cc0152000001223300600200123300500148000008004888c8ccd54c0bc48004d401d40148d4d5413000488ccd54c0c84800540108d4d5413c00488c028004004cc06400c0084d401940104cd4018004104894cd4c1000084004410448cd40cc88ccd401400c008004d400800448d4d400c0048800448d4d40080048800848848cc00400c00848004800480044cd40ad4009400448004488cd55400c008004444888c00cc008004888c8c8c8c004018d4004800448c004d5411888cd4d40f400520002235355042002225335303d333573466e3c0080280fc0f84c0200044c01800c8c8c8c00400cd4004800448c004d5411888cd4d40f400520002235355042002225335303d333573466e3c0080240fc0f840044c01800c8d4c0d40048880084cd4095400540d84c00c04c4cd403400540d088ccc00c04c008004888cd54c024480048d4d540dc00488cd540e8008cd54c030480048d4d540e800488cd540f4008ccd4d540540048cc0292000001223300b00200123300a00148000004cc01000800488cd54c01c480048d4d540d400488cd540e0008ccd4d540400048cd54c02c480048d4d540e400488cd540f0008d5405c00400488ccd5540200bc0080048cd54c02c480048d4d540e400488cd540f0008d54054004004ccd55400c0a8008004444888ccd54c0544800540c8cd54c01c480048d4d540d400488cd540e0008d5404c004ccd54c0544800488d4d540d8008894cd4c0c4ccd54c07048004d402140348d4d540e400488cc028008014018400c4cd40d801000d40cc004cd54c01c480048d4d540d400488c8c8cd540e8010c004018d4004800448c004d54100894cd4d40dc0044d54050010884d4d540f0008894cd4c0dccc0340080244cd540640200044c01800c00848cd407c88ccd401400c008004d400800448d4d401c0048800448d4d401800488008d4004800448c004d540d48844894cd4d40b8004540c0884cd40c4c010008cd54c018480040100044cd400c004094894cd4c094008409c400448848cc00400c008480044484888c00c01044884888cc0080140104484888c00401044800488cdc00010009299a9a8131803000909a980d800911a98110009111a9808803911a980a0011111111111199aa980b09000911a9819801111299a98199981600a001899a81c0028020802281a00489a80a1a980119ab9c491024c6600003498480048004d4004800448c004d5409c88448894cd4d40840044008884cc014008ccd54c01c480040140100048d4c01800488d4c02400888888888894cccd4c05002c8540bc8540bc8540bc84ccd54c03c48005405c8d4c08c004894cd4c09ccc0780080104d40cc00c540c802cd4004800448c004d5409088448c8894cd4d407c0044d4020010884cd4014c010008ccd54c020480040180100044d401800448d4d40200048800448d4d401c0048800880048004800480044cd4009400d406048848cc00400c008480044894cd4d40580088400c4004894cd4c02cccd5cd19b8f35300a0022200235300a0012200200d00c1333573466e1cd4c02800888004d4c02800488004034030403049888d4c04800888d4c05000c88c8cd4c0700148cd4c07401094cd4c040ccd5cd19b8f00200101201115003101120112335301d0042011253353010333573466e3c0080040480445400c404454cd4d405c00c854cd4d406000884cc024008004403c54cd4d405c0048403c403c88cd4c0540088cd4c0580088cc018008004888034888cd4c06001080348894cd4c038ccd5cd19b8700600301000f15335300e333573466e1c01400804003c4cc024010004403c403c88ccd5cd19b8700200100900822335301400223353015002233005002001200923353015002200923300500200122333573466e3c00800401c01880048004488008488004800480044488008488488cc00401000c48004448848cc00400c008448004800448488c00800c44880044800480048004448c8c00400488cc00cc008008004cc88ccc888cc88ccc008cd5401d22011c469f9a74824555709042fb95aa2d11e3c7fa4eada2fe97bc287687dd004881044641524d0033550074891c150ee39332bddb6083c2294cedb206aadbf7606e4bf53184fc7e9cad004881065049475354590033500433550074891c8bb3b343d8e404472337966a722150048c768d0a92a9813596c5338d00335004335500748905745049475900480514015401488848ccc00401000c00880044488008488488cc00401000c48004448848cc00400c0084480041"
                  }
                },
                "datums": {
                  "5865d96e4313780a37af26055cdcc90308db23adaf4f2082178355e5e1dc20f6": "a54a63757272656e63696573a246736f75726365581c68747470733a2f2f6e6f746e756c6c736f6c7574696f6e732e636f6d4773796d626f6c739fa4457363616c651a000186a04673796d626f6c4345555244756e6974474555522f5553444576616c75651a00014adca4457363616c651a000186a04673796d626f6c4347425044756e6974474742502f5553444576616c75651a00011c15a4457363616c6518644673796d626f6c4349445244756e6974474944522f5553444576616c75651a0015b625ff466d6574616c73a246736f75726365581c68747470733a2f2f6e6f746e756c6c736f6c7574696f6e732e636f6d4773796d626f6c739fa4457363616c6518644673796d626f6c42417544756e6974495553442f6f756e63654576616c75651a0002c528a4457363616c6518644673796d626f6c42416744756e6974495553442f6f756e63654576616c756519095ca4457363616c65014673796d626f6c42507444756e6974495553442f6f756e63654576616c7565190400a4457363616c65014673796d626f6c42506444756e6974495553442f6f756e63654576616c75651909b3ff4773657276696365581c68747470733a2f2f6f7261636c652e70696779746f6b656e2e636f6d44736f6672a246736f75726365583768747470733a2f2f7777772e6e6577796f726b6665642e6f72672f6d61726b6574732f7265666572656e63652d72617465732f736f66724773796d626f6c739fa4457363616c6518644673796d626f6c44534f465244756e697441254576616c756505ff4974696d657374616d705819323032312d30382d33315431373a30363a31352b30303a3030"
                },
                "redeemers": {
                  "spend:1": {
                    "redeemer": "02",
                    "executionUnits": {
                      "memory": 4319292,
                      "steps": 1448909354
                    }
                  }
                },
                "bootstrap": []
              },
              "raw": "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dmFsdWUZCbP/R3NlcnZpY2VYHGh0dHBzOi8vb3JhY2xlLnBpZ3l0b2tlbi5jb21Ec29mcqJGc291cmNlWDdodHRwczovL3d3dy5uZXd5b3JrZmVkLm9yZy9tYXJrZXRzL3JlZmVyZW5jZS1yYXRlcy9zb2ZyR3N5bWJvbHOfpEVzY2FsZRhkRnN5bWJvbERTT0ZSRHVuaXRBJUV2YWx1ZQX/SXRpbWVzdGFtcFgZMjAyMS0wOC0zMVQxNzowNjoxNSswMDowMAWBhAABAoIaAEHoPBpWXJoq9dkBA6EAoRoAA8aEpWpjdXJyZW5jaWVzomZzb3VyY2V4HGh0dHBzOi8vbm90bnVsbHNvbHV0aW9ucy5jb21nc3ltYm9sc4OkZXNjYWxlGgABhqBmc3ltYm9sY0VVUmR1bml0Z0VVUi9VU0RldmFsdWUaAAFJ9qRlc2NhbGUaAAGGoGZzeW1ib2xjR0JQZHVuaXRnR0JQL1VTRGV2YWx1ZRoAARu4pGVzY2FsZRhkZnN5bWJvbGNJRFJkdW5pdGdJRFIvVVNEZXZhbHVlGgAVvf9mbWV0YWxzomZzb3VyY2V4HGh0dHBzOi8vbm90bnVsbHNvbHV0aW9ucy5jb21nc3ltYm9sc4SkZXNjYWxlGGRmc3ltYm9sYkF1ZHVuaXRpVVNEL291bmNlZXZhbHVlGgACxHekZXNjYWxlGGRmc3ltYm9sYkFnZHVuaXRpVVNEL291bmNlZXZhbHVlGQlwpGVzY2FsZQFmc3ltYm9sYlB0ZHVuaXRpVVNEL291bmNlZXZhbHVlGQPvpGVzY2FsZQFmc3ltYm9sYlBkZHVuaXRpVVNEL291bmNlZXZhbHVlGQmTZW55ZmVkomZzb3VyY2V4N2h0dHBzOi8vd3d3Lm5ld3lvcmtmZWQub3JnL21hcmtldHMvcmVmZXJlbmNlLXJhdGVzL3NvZnJnc3ltYm9sc4GkZXNjYWxlGGRmc3ltYm9sZFNPRlJkdW5pdGElZXZhbHVlBWdzZXJ2aWNleBxodHRwczovL29yYWNsZS5waWd5dG9rZW4uY29taXRpbWVzdGFtcHgZMjAyMS0wOS0wMVQyMDo0NDozMiswMDowMA==",
              "id": "d0cca8b82263a4c1ad2c4f845ab58ed5610fdb1fcbe76faa92d19ef5aa2655b2",
              "body": {
                "inputs": [],
                "collaterals": [
                  {
                    "txId": "9fca96034e904d0a5c9cef27fa83188da4169ccdaefde0c2518e5f093a43dee3",
                    "index": 3
                  }
                ],
                "outputs": [
                  {
                    "address": "addr_test1qq2el3jxkkg6g3euz9ymzs35kg08drlu33w5uharfx6g8x28gu262anw6rvqr5th53h8khjhkwrdzyq3qercygw3z6yq6vd6w0",
                    "value": {
                      "coins": 100547906,
                      "assets": {}
                    },
                    "datumHash": null,
                    "datum": null
                  },
                  {
                    "address": "addr_test1wquuc74u5r702y8jpazgm3nusse6jaj68cm2xqyzyqhyu8g25ysjg",
                    "value": {
                      "coins": 5000000,
                      "assets": {
                        "150ee39332bddb6083c2294cedb206aadbf7606e4bf53184fc7e9cad.504947535459": 1
                      }
                    },
                    "datumHash": "a9bb6999d768e9e04bae81970e39c3b496c756adbdd1b137bfdba5a9ef3678a0",
                    "datum": "a9bb6999d768e9e04bae81970e39c3b496c756adbdd1b137bfdba5a9ef3678a0"
                  },
                  {
                    "address": "addr_test1qq2el3jxkkg6g3euz9ymzs35kg08drlu33w5uharfx6g8x28gu262anw6rvqr5th53h8khjhkwrdzyq3qercygw3z6yq6vd6w0",
                    "value": {
                      "coins": 103384646,
                      "assets": {
                        "469f9a74824555709042fb95aa2d11e3c7fa4eada2fe97bc287687dd.4641524d": 1
                      }
                    },
                    "datumHash": null,
                    "datum": null
                  },
                  {
                    "address": "addr_test1qq2el3jxkkg6g3euz9ymzs35kg08drlu33w5uharfx6g8x28gu262anw6rvqr5th53h8khjhkwrdzyq3qercygw3z6yq6vd6w0",
                    "value": {
                      "coins": 2000000,
                      "assets": {}
                    },
                    "datumHash": null,
                    "datum": null
                  }
                ],
                "certificates": [],
                "withdrawals": {},
                "fee": 836739,
                "validityInterval": {
                  "invalidBefore": null,
                  "invalidHereafter": null
                },
                "mint": {
                  "coins": 0,
                  "assets": {}
                },
                "network": null,
                "scriptIntegrityHash": "377ce1c5e344b4ba99f56c23619e498088b7154c1f387e467cdd94774256ec90",
                "requiredExtraSignatures": []
              },
              "inputSource": "inputs"
            }
          ],
          "header": {
            "signature": "WzXJKK5w7r6PiplnbgHruL3vPBsp25GjgNMKEG+i2yNdJyHcIrQ7Gy7/B8KMXNQhIU1kAFarVpfKcEf6oN6pDk48rIqp2V1ytQmn6hqvXiLYTW/Ee7l50Dist3rvKlztK0Z75B5v49+PU5ollX5wmlXOB9EHGRsPhm0M/IXC56+DS8/EcaSTClJWkAuA48qSBj9r0t9ZM219QZONcliFPksOr2uMwIu7+1HKTiMblTCang3OMnKM9INsVGjWD1BUtiBsuWRxTMnvUclgefrCSX1EqE/kmh3DhSWngVZvQ6QsjqiKloBgGY0k/aIOdUVqvCywPpETbCbVYeIKLwje5WL577dftAdeFGkm35oIV1wTh4L3ynkbLkYULIw74qVdbud98azKYG75aC8NVvH5qtQNDOdWqgHdazLO6L9KzgUXN8gM9l8wze5z83OezKDZl3pEXw5pP2x/5qPWzmosIRqH8URsl2Tv41sliju93Hg53Pn+ABCtvkxCGeJePkqX0UY7FyjCuGm1Q8BuUrRwAYTvMZyIq690TQBws6YSPzlpZjvROSvhQG1eh+CRnMgT4zPkMaVwtA67rDS7+mEGwQ==",
            "nonce": {
              "output": "4027vOEqHhktDnDyyK15RhzP0uekYHz3igwRn0sv7vZ/FqVVHimdVb/6UiOCnUvm+cB8GISXtvuWNO/x5jwZPg==",
              "proof": "1yM2OCYdaiipW9ibEdfYGfylVWjXdYIWu/1edLoBuoNdEwV1xIcUU1BaEXtWXI9bWsXanOhzmH88/iul23jEnVIdAa1pbdEs2RT8bxRp/gE="
            },
            "leaderValue": {
              "output": "AB3gWlYr0Bf+2uOECAlVMN6UD3Ghuw38bBQS8RIyJIr23Vl1iezZgC63vpt6yHhxzYLq+K0DqmjxA/0EXljPiw==",
              "proof": "xpyQWyI3ny7Ll0aNpnxVnmn1GWMqdjYXU5hFW0g791JCw7C3XQPHzbPmmap0ypwDOJwySe21aJ3d2FMkNoYHlxLE6sRU9K55oRvjuS/lJQ8="
            },
            "opCert": {
              "hotVk": "T3uhokqx6BYURJC99X7g6c3wqmjYYV6787uXrdWVcIw=",
              "count": 5,
              "kesPeriod": 268,
              "sigma": "ifaVk3UPIEM/I3lJvEwNKjyZL6BgYRHDyqJNNrqyh75pKYnCLkpUvxTLLTWXo7nQZQlm03KkQ1NbmoBeHxA6BA=="
            }
          },
          "headerHash": "8add217cbfe546c3aa937bd4a6dddaa16ddc57198ecbcdaff39b2b8c7ffa6296"
        }
      }
    }
  },
  "reflection": null
}
`
	var response Response
	err := json.Unmarshal([]byte(data), &response)
	if err != nil {
		t.Fatalf("error unmarshalling response: %v", err)
	}
}

func TestVasil_DatumParsing_Base64(t *testing.T) {
	data := `{"datums": {"a": "2HmfWBzIboNaGwk6qBYQ/Tk19GPOUpkpze2Ldfe1HOZEQpwK/w=="}}`
	var response Witness
	err := json.Unmarshal([]byte(data), &response)
	if err != nil {
		t.Fatalf("error unmarshalling response: %v", err)
	}

	datumHex := response.Datums["a"]
	_, err = hex.DecodeString(datumHex)
	if err != nil {
		t.Fatalf("error decoding hex string: %v", err)
	}
}

func TestVasil_DatumParsing_Hex(t *testing.T) {
	data := `{"datums": {"a": "d8799f581cc86e835a1b093aa81610fd3935f463ce529929cded8b75f7b51ce644429c0aff"}}`
	var response Witness
	err := json.Unmarshal([]byte(data), &response)
	if err != nil {
		t.Fatalf("error unmarshalling response: %v", err)
	}

	datumHex := response.Datums["a"]
	_, err = hex.DecodeString(datumHex)
	if err != nil {
		t.Fatalf("error decoding hex string: %v", err)
	}
}

func TestVasil_BackwardsCompatibleWithExistingDynamoDB(t *testing.T) {
	data, err := os.ReadFile("testdata/scoop.json")
	assert.Nil(t, err)

	var item map[string]*dynamodb.AttributeValue
	err = json.Unmarshal(data, &item)
	assert.NoError(t, err)

	var response Tx
	err = dynamodbattribute.Unmarshal(item["tx"], &response)
	assert.NoError(t, err)
	fmt.Println(response.Witness.Datums)
}

func TestValue_Equals(t *testing.T) {
	assert.True(t, Equals(Value{Coins: num.Uint64(0)}, Value{Coins: num.Uint64(0)}))
	assert.True(t, Equals(
		Value{Coins: num.Uint64(1), Assets: map[AssetID]num.Int{"A": num.Uint64(10), "B": num.Uint64(0)}},
		Value{Coins: num.Uint64(1), Assets: map[AssetID]num.Int{"A": num.Uint64(10), "B": num.Uint64(0)}},
	))
	assert.True(t, Equals(
		Value{Coins: num.Uint64(1), Assets: map[AssetID]num.Int{"A": num.Uint64(10), "B": num.Uint64(15)}},
		Value{Coins: num.Uint64(1), Assets: map[AssetID]num.Int{"A": num.Uint64(10), "B": num.Uint64(15)}},
	))
	assert.False(t, Equals(Value{Coins: num.Uint64(0)}, Value{Coins: num.Uint64(1)}))
	assert.False(t, Equals(
		Value{Coins: num.Uint64(1), Assets: map[AssetID]num.Int{"A": num.Uint64(10), "B": num.Uint64(0)}},
		Value{Coins: num.Uint64(1)},
	))
	assert.False(t, Equals(
		Value{Coins: num.Uint64(1), Assets: map[AssetID]num.Int{"A": num.Uint64(10), "B": num.Uint64(10)}},
		Value{Coins: num.Uint64(1), Assets: map[AssetID]num.Int{"A": num.Uint64(10), "B": num.Uint64(15)}},
	))
}
