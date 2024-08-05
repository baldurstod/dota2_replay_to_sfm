package main_test

import (
	"encoding/json"
	"errors"
	"log"
	"math"
	"os"
	"path"
	"strings"
	"testing"

	replay "github.com/baldurstod/dota2_replay_to_sfm"
	"github.com/baldurstod/dota2_replay_to_sfm/entities"
	dota_items "github.com/baldurstod/go-dota2"
	"github.com/baldurstod/go-sfm"
	"github.com/baldurstod/go-sfm/utils"
	"github.com/baldurstod/go-sfm/utils/dota2"
	"github.com/baldurstod/go-vector"
	"github.com/baldurstod/manta"
	"github.com/baldurstod/manta/dota"
)

const DEG_TO_RAD = math.Pi / 180

var characters = func() map[uint64]*sfm.AnimationSet { return make(map[uint64]*sfm.AnimationSet) }()
var characters2 = func() map[uint64]*dota2.Character { return make(map[uint64]*dota2.Character) }()
var itemsPerPlayer = func() map[uint64]map[uint32]struct{} { return make(map[uint64]map[uint32]struct{}) }()
var clip *sfm.FilmClip

func TestReplay(t *testing.T) {

	a := replay.NewReplay()
	/*
		wearableItems := make([]*entities.DOTAWearableItem, 0, 100)
		playerControllersByAccountId := make(map[uint64]*entities.DOTAPlayerController)
		playerControllersByHandle := make(map[uint64]*entities.DOTAPlayerController)
	*/

	// Create a new parser instance from a file. Alternatively see NewParser([]byte)
	filename := "./var/7865917356.dem"
	//filename = "./var/7865849382.dem"
	//filename = "./var/7382065860_1966034883.dem"
	f, err := os.Open(filename)
	if err != nil {
		log.Fatalf("unable to open file: %s", err)
	}
	defer f.Close()

	p, err := manta.NewStreamParser(f)
	if err != nil {
		log.Fatalf("unable to create parser: %s", err)
	}

	p.Callbacks.OnCDemoClassInfo(func(m *dota.CDemoClassInfo) error {
		return nil
	})
	p.Callbacks.OnCDemoAnimationData(func(animData *dota.CDemoAnimationData) error {
		log.Println(animData)
		return nil
	})

	//entities := make(map[string]*manta.Entity)
	entitiesj := make(map[string]map[string]any)
	//entities2 := make(map[uint64]map[string]any)
	entities2 := make([]any, 0, 1000)
	if err := initSession(); err != nil {
		t.Error(err)
		return
	}

	firstTick := int32(2000)
	const INDEX_BITS = 14

	p.OnEntity(func(e *manta.Entity, op manta.EntityOp) error {

		if int32(p.Tick)-firstTick > 2800 {
			return nil
		}
		handle := uint64((e.GetSerial() << INDEX_BITS) | e.GetIndex())
		m := e.Map()
		className := e.GetClassName()
		switch className {
		case "CDOTAWearableItem":
			/*
				wearable := &entities.DOTAWearableItem{}
				wearable.InitFromEntity(m)
				wearableItems = append(wearableItems, wearable)
			*/
			a.AddWearable(m)
		case "CDOTAPlayerController":
			/*
				controller := &entities.DOTAPlayerController{}
				controller.InitFromEntity(m, handle)
				playerControllersByAccountId[controller.SteamID-BASE_STEAM_ID] = controller
				playerControllersByHandle[controller.Handle] = controller
			*/
			a.AddPlayerController(m, handle)
		}

		if strings.HasPrefix(className, "CDOTA_Unit_Hero_") {
			unitHero := &entities.DOTAUnitHero{}
			unitHero.InitFromEntity(m)
			//playercontrollers[controller.SteamID] = controller
			a.AddUnit(m, handle)
		}

		//if strings.HasPrefix(className, "CDOTA_Ability_") {
		if strings.HasPrefix(className, "CDynamicProp") {
			idx, found := m["m_pEntity.m_nameStringableIndex"]
			if found {
				name, found := p.LookupStringByIndex("EntityNames", idx.(int32))
				log.Println(e.GetClassName(), name, found)
			}
		}

		//"CDOTATeam"
		//log.Println(e, op)
		if int32(p.Tick) < firstTick {
			return nil
		}
		m["tick"] = p.Tick
		m["index"] = e.GetIndex()
		m["serial"] = e.GetSerial()
		m["handle"] = handle

		_, exist := entitiesj[className]
		if !exist {
			entitiesj[className] = m
		}
		if className == "CDOTAWearableItem" {
			entities2 = append(entities2, map[string]any{"key": className, "value": m})
			//entities2[m["CBodyComponent.m_hModel"].(uint64)] = m
		}
		/*
			if className == "CDOTAWearableItem" {
				var accountID, itemDefinitionIndex any
				var found bool
				if accountID, found = m["m_iAccountID"]; !found {
					return nil
				}
				if itemDefinitionIndex, found = m["m_iItemDefinitionIndex"]; !found {
					return nil
				}
				def := itemDefinitionIndex.(uint32)

				if acc := accountID.(uint64); acc != 0 {
					steamId := acc + BASE_STEAM_ID
					if _, found = itemsPerPlayer[steamId]; !found {
						itemsPerPlayer[steamId] = make(map[uint32]struct{})
					}

					itemsPerPlayer[steamId][def] = struct{}{}
				}
			}
		*/

		if idx, found := m["m_pEntity.m_nameStringableIndex"]; found && strings.HasPrefix(className, "CDOTA_Unit_Hero_") {
			//log.Println(a.GetItems(handle))

			var ownerEntity any
			if ownerEntity, found = m["m_hOwnerEntity"]; !found {
				return nil
			}

			name, found := p.LookupStringByIndex("EntityNames", idx.(int32))
			if !found {
				return nil
			}

			c, err := getCharacter(ownerEntity.(uint64), name)
			if err != nil {
				t.Error(err)
				return nil
			}
			//log.Println(c)

			//if c.GetEntity() == "npc_dota_hero_puck" {
			for _, item := range a.GetItems(handle) {
				c.EquipItem(item)
			}
			//}

			as, err := createCharacterModel(ownerEntity.(uint64), name)
			if err != nil {
				t.Error(err)
				return nil
			}

			//log.Println(e, p)
			if firstTick == 0 {
				firstTick = int32(p.Tick)
			}

			tc := as.GetTransformControl(sfm.ROOT_TRANSFORM)
			var posLayer *sfm.LogLayer[vector.Vector3[float32]]
			var rotLayer *sfm.LogLayer[vector.Quaternion[float32]]
			posLayer = any(tc.PositionChannel.Log.GetLayer("vector3 log")).(*sfm.LogLayer[vector.Vector3[float32]])
			rotLayer = any(tc.OrientationChannel.Log.GetLayer("quaternion log")).(*sfm.LogLayer[vector.Quaternion[float32]])

			time := float32(int32(p.Tick)-firstTick) / 30.

			// TODO -128 should be replaced by "CWorld".CBodyComponent.m_cellX
			posLayer.SetValue(time, vector.Vector3[float32]{
				(float32(m["CBodyComponent.m_cellX"].(uint64))-128)*128. + m["CBodyComponent.m_vecX"].(float32),
				(float32(m["CBodyComponent.m_cellY"].(uint64))-128)*128. + m["CBodyComponent.m_vecY"].(float32),
				(float32(m["CBodyComponent.m_cellZ"].(uint64))-128)*128. + m["CBodyComponent.m_vecZ"].(float32),
			})

			rot := m["CBodyComponent.m_angRotation"].([]float32)
			q := vector.Quaternion[float32]{}
			q.FromEuler(rot[0]*DEG_TO_RAD, rot[2]*DEG_TO_RAD, rot[1]*DEG_TO_RAD)

			rotLayer.SetValue(time, q)
		}

		return nil
	})

	//count := 0
	/*
		p.OnEntity(func(e *manta.Entity, op manta.EntityOp) error {
			if true {

				idx, found := e.Map()["m_pEntity.m_nameStringableIndex"]
				if !found {
					return nil
				}
				//log.Println(idx.(int32))
				if !found || idx.(int32) == -1 {
					return nil
				}
				name, found := p.LookupStringByIndex("EntityNames", idx.(int32))
				//log.Println(e.Map(), idx, name)
				log.Println(e.GetClassName(), name, found)
				if name == "npc_dota_base" {
					//count++
					log.Println(e)
				}
			}
			return nil
		})
	*/

	// Start parsing the replay!
	log.Printf("Start parsing\n")
	if err := p.Start(); err != nil {
		t.Error(err)
		return
	}

	//log.Println(entities)
	log.Println("Parse Complete!")

	for _, character := range characters2 {
		if character == nil {
			continue
		}
		if err = character.CreateItemModels(clip); err != nil {
			t.Error(err)
			return
		}
	}

	js := make(map[string]any)
	js["entities"] = entitiesj
	js["entities2"] = entities2
	lookup := make([]string, 0, 1000) //p.LookupStringByIndex("EntityNames")
	log.Println(itemsPerPlayer)

	for i := int32(0); ; i++ {
		if s, ok := p.LookupStringByIndex("EntityNames", i); ok {
			lookup = append(lookup, s)
		} else {
			break
		}
	}

	js["lookup"] = lookup

	j, _ := json.MarshalIndent(js, "", "\t")
	os.WriteFile(path.Join(varFolder, "entities.json"), j, 0666)

	writeSession(t)
}

/*
CParticleSystem
CDynamicProp
*/

func initDota() error {
	buf, err := os.ReadFile(varFolder + "npc_heroes.txt")
	if err != nil {
		return err
	}
	err = dota_items.InitHeroes(buf)
	if err != nil {
		return err
	}

	buf, err = os.ReadFile(varFolder + "items_game.txt")
	if err != nil {
		return err
	}
	err = dota_items.InitItems(buf)
	if err != nil {
		return err
	}
	return nil
}

var session *sfm.Session

func initSession() error {

	if err := initDota(); err != nil {
		return err
	}

	session = sfm.NewSession()

	clip = utils.CreateClip(session)

	clip.Camera.Transform.Orientation.RotateZ(math.Pi)
	clip.Camera.Transform.Position.Set(200, 0, 150)

	//shot1.MapName = "maps/dota.vmap"
	return nil

}

func getCharacter(owner uint64, name string) (*dota2.Character, error) {
	if c, exist := characters2[owner]; exist {
		return c, nil
	}

	c, err := dota2.NewCharacter(name)
	if err != nil {
		return nil, err
	}
	characters2[owner] = c
	return c, nil
}

func createCharacterModel(owner uint64, name string) (*sfm.AnimationSet, error) {
	if c, exist := characters[owner]; exist {
		return c, nil
	}

	as, err := characters2[owner].CreateGameModel(clip)
	if err != nil {
		return nil, err
	}

	tc := as.GetTransformControl(sfm.ROOT_TRANSFORM)
	//var posLayer *sfm.LogLayer[vector.Vector3[float32]]
	//var rotLayer *sfm.LogLayer[vector.Quaternion[float32]]
	if tc == nil {
		return nil, errors.New("unable to get rootTransform")
	}

	//posLayer = any(tc.PositionChannel.Log.GetLayer("vector3 log")).(*sfm.LogLayer[vector.Vector3[float32]])
	//rotLayer = any(tc.OrientationChannel.Log.GetLayer("quaternion log")).(*sfm.LogLayer[vector.Quaternion[float32]])

	/*
		s2Model, err := GetModel("dota2", model.ModelName)
		if err != nil {
			return err
		}

		seq, err := s2Model.GetSequenceByName(animation)
		if err != nil {
			return err
		}*/

	/*
		if posLayer == nil {
			t.Error("posLayer == nil")
			return
		}
		if rotLayer == nil {
			t.Error("rotLayer == nil")
			return
		}*/
	/*
		if err = utils.PlaySequence(as, "idle", clip.GetDuration()); err != nil {
			return nil, err
		}
	*/
	characters[owner] = as

	return as, nil
}

func writeSession(t *testing.T) {
	err := session.WriteTextFile(path.Join(varFolder, "test_session.dmx"))
	if err != nil {
		t.Error(err)
		return
	}
}

func TestTick(t *testing.T) {
	// Create a new parser instance from a file. Alternatively see NewParser([]byte)
	filename := "./var/7865849382.dem"
	f, err := os.Open(filename)
	if err != nil {
		log.Fatalf("unable to open file: %s", err)
	}
	defer f.Close()

	p, err := manta.NewStreamParser(f)
	if err != nil {
		log.Fatalf("unable to create parser: %s", err)
	}

	log.Println("start")
	log.Println(p.GetLastTick())
}

func getPlayerByAccountId(owner uint64, name string) (*entities.DOTAPlayerController, error) {
	return nil, nil
}
