package main_test

import (
	"encoding/json"
	"log"
	"math"
	"os"
	"path"
	"testing"

	dota_items "github.com/baldurstod/go-dota2"
	"github.com/baldurstod/go-sfm"
	"github.com/baldurstod/go-sfm/utils"
	"github.com/baldurstod/go-sfm/utils/dota2"
	"github.com/baldurstod/go-vector"
	"github.com/dotabuff/manta"
	"github.com/dotabuff/manta/dota"
)

func TestReplay(t *testing.T) {
	// Create a new parser instance from a file. Alternatively see NewParser([]byte)
	filename := "./var/7865917356.dem"
	filename = "./var/7382065860_1966034883.dem"
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
	entities := make(map[string]map[string]any)
	as := initSession(t)
	if as == nil {
		return
	}

	tc := as.GetTransformControl("rootTransform")
	var layer *sfm.LogLayer[vector.Vector3[float32]]
	if tc != nil {
		layer = any(tc.PositionChannel.Log.GetLayer("vector3 log")).(*sfm.LogLayer[vector.Vector3[float32]])
	}

	if layer == nil {
		t.Error("layer == nil")
		return
	}

	firstTick := uint32(0)

	p.OnEntity(func(e *manta.Entity, op manta.EntityOp) error {
		//log.Println(e, op)

		if p.Tick < 50000 {
			return nil

		}
		if (firstTick > 0) && (p.Tick-firstTick) > 2000 {
			return nil
		}

		m := e.Map()
		_, exist := entities[e.GetClassName()]
		if !exist {
			entities[e.GetClassName()] = m
		}

		if idx, found := m["m_pEntity.m_nameStringableIndex"]; found {
			if idx.(int32) == 490 {
				//log.Println(e, p)
				if firstTick == 0 {
					firstTick = p.Tick
				}

				layer.SetValue(float32(p.Tick-firstTick)/30., vector.Vector3[float32]{
					(float32(m["CBodyComponent.m_cellX"].(uint64))-128)*128. + m["CBodyComponent.m_vecX"].(float32),
					(float32(m["CBodyComponent.m_cellY"].(uint64))-128)*128. + m["CBodyComponent.m_vecY"].(float32),
					(float32(m["CBodyComponent.m_cellZ"].(uint64))-128)*128. + m["CBodyComponent.m_vecZ"].(float32),
				})
			}
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
	p.Start()

	//log.Println(entities)
	log.Println("Parse Complete!")

	js := make(map[string]any)
	js["entities"] = entities
	lookup := make([]string, 0, 1000) //p.LookupStringByIndex("EntityNames")

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

func initSession(t *testing.T) *sfm.AnimationSet {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	if err := initDota(); err != nil {
		t.Error(err)
		return nil
	}

	c, err := dota2.NewCharacter("npc_dota_hero_dark_willow")
	if err != nil {
		t.Error(err)
		return nil
	}

	session = sfm.NewSession()

	shot1, err := utils.CreateClip(session)
	if err != nil {
		t.Error(err)
		return nil
	}

	shot1.Camera.Transform.Orientation.RotateZ(math.Pi)
	shot1.Camera.Transform.Position.Set(200, 0, 150)
	shot1.Camera.ZFar = 50000

	as, err := c.CreateGameModel(shot1)
	if err != nil {
		t.Error(err)
		return nil
	}

	err = utils.PlaySequence(as, "idle", shot1.GetDuration())
	if err != nil {
		t.Error(err)
		return nil
	}
	return as
}

func writeSession(t *testing.T) {
	err := session.WriteTextFile(path.Join(varFolder, "test_session.dmx"))
	if err != nil {
		t.Error(err)
		return
	}
}
