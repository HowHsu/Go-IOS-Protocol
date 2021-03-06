package pob

import (
	"testing"

	"github.com/iost-official/Go-IOS-Protocol/account"
	. "github.com/iost-official/Go-IOS-Protocol/account"
	. "github.com/iost-official/Go-IOS-Protocol/consensus/common"
	"github.com/iost-official/Go-IOS-Protocol/core/block"
	. "github.com/smartystreets/goconvey/convey"
)

func TestGlobalStaticProperty(t *testing.T) {
	Convey("Test of witness lists of static property", t, func() {
		prop := newGlobalStaticProperty(
			Account{
				ID:     "id0",
				Pubkey: []byte{},
				Seckey: []byte{},
			},
			[]string{"id1", "id2", "id3"},
		)
		Convey("New", func() {
			So(prop.NumberOfWitnesses, ShouldEqual, 3)
		})

		prop.addPendingWitness("id4")
		prop.addPendingWitness("id5")
		Convey("Add pending witness", func() {
			So(len(prop.PendingWitnessList), ShouldEqual, 2)
			err := prop.addPendingWitness("id4")
			So(err, ShouldNotBeNil)
		})

		Convey("Update lists", func() {
			prop.updateWitnessLists([]string{"id3", "id5", "id1"})
			So(prop.WitnessList[0], ShouldEqual, "id1")
			So(prop.WitnessList[1], ShouldEqual, "id3")
			So(prop.WitnessList[2], ShouldEqual, "id5")
			So(prop.PendingWitnessList[0], ShouldEqual, "id2")
			So(prop.PendingWitnessList[1], ShouldEqual, "id4")
		})

		Convey("Delete pending witness", func() {
			err := prop.deletePendingWitness("id4")
			So(len(prop.PendingWitnessList), ShouldEqual, 1)
			So(err, ShouldBeNil)

			err = prop.deletePendingWitness("id2")
			So(err, ShouldNotBeNil)
		})
	})
}

func TestGlobalDynamicProperty(t *testing.T) {
	Convey("Test of global dynamic property", t, func() {
		sp := newGlobalStaticProperty(
			account.Account{
				ID:     "id1",
				Pubkey: []byte{},
				Seckey: []byte{},
			},
			[]string{"id0", "id1", "id2"},
		)
		dp := newGlobalDynamicProperty()
		dp.LastBlockNumber = 0
		dp.TotalSlots = 0
		dp.LastBlockTime = Timestamp{Slot: 0}
		startTs := Timestamp{Slot: 70002}
		bh := block.BlockHead{
			Number:  1,
			Time:    startTs.Slot,
			Witness: "id0",
		}
		dp.update(&bh)

		Convey("update first block", func() {
			So(dp.LastBlockNumber, ShouldEqual, 1)
		})

		curSec := startTs.ToUnixSec() + 1
		sec := timeUntilNextSchedule(&sp, &dp, curSec)
		Convey("in other's slot", func() {
			curTs := GetTimestamp(curSec)
			wit := witnessOfTime(&sp, &dp, curTs)
			So(wit, ShouldEqual, "id0")
			So(sec, ShouldBeLessThanOrEqualTo, SlotLength)
		})

		curSec += SlotLength - 1
		timestamp := GetTimestamp(curSec)
		Convey("in self's slot", func() {
			wit := witnessOfTime(&sp, &dp, timestamp)
			So(wit, ShouldEqual, "id1")
			wit = witnessOfSec(&sp, &dp, curSec)
			So(wit, ShouldEqual, "id1")
		})

		bh.Number = 2
		bh.Time = timestamp.Slot
		bh.Witness = "id1"
		dp.update(&bh)
		Convey("update second block", func() {
			So(dp.LastBlockNumber, ShouldEqual, 2)
		})

		curSec += 1
		sec = timeUntilNextSchedule(&sp, &dp, curSec)
		Convey("in self's slot, but finished", func() {
			So(sec, ShouldBeGreaterThanOrEqualTo, SlotLength*2)
			So(sec, ShouldBeLessThanOrEqualTo, SlotLength*3)
		})

		curSec += SlotLength*3 - 1
		Convey("in self's slot and lost two previous blocks", func() {
			curTs := GetTimestamp(curSec)
			wit := witnessOfTime(&sp, &dp, curTs)
			So(wit, ShouldEqual, "id1")
			wit = witnessOfSec(&sp, &dp, curSec)
			So(wit, ShouldEqual, "id1")
		})

		timestamp = GetTimestamp(curSec)
		bh.Number = 3
		bh.Time = timestamp.Slot
		bh.Witness = "id1"
		dp.update(&bh)
		Convey("update third block", func() {
			So(dp.LastBlockNumber, ShouldEqual, 3)
		})
	})
}
