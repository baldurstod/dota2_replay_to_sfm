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

	dota_items "github.com/baldurstod/go-dota2"
	"github.com/baldurstod/go-sfm"
	"github.com/baldurstod/go-sfm/utils"
	"github.com/baldurstod/go-sfm/utils/dota2"
	"github.com/baldurstod/go-vector"
	"github.com/baldurstod/manta"
	"github.com/baldurstod/manta/dota"
)

const DEG_TO_RAD = math.Pi / 180

var characters = func() map[string]*sfm.AnimationSet { return make(map[string]*sfm.AnimationSet) }()
var clip *sfm.FilmClip

func TestReplay(t *testing.T) {
	// Create a new parser instance from a file. Alternatively see NewParser([]byte)
	filename := "./var/7865917356.dem"
	filename = "./var/7865849382.dem"
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
	entities := make(map[string]map[string]any)
	if err := initSession(); err != nil {
		t.Error(err)
		return
	}

	firstTick := uint32(0)

	p.OnEntity(func(e *manta.Entity, op manta.EntityOp) error {
		className := e.GetClassName()
		if !strings.HasPrefix(className, "CDOTA_Unit_Hero_") {
			return nil

		}

		//log.Println(e, op)
		/*
			if p.Tick < 2000 {
				return nil
			}
		*/
		/*
			if p.Tick-firstTick > 1800 {
				return nil
			}
		*/

		m := e.Map()
		_, exist := entities[className]
		if !exist {
			entities[className] = m
		}

		if idx, found := m["m_pEntity.m_nameStringableIndex"]; found {

			name, found := p.LookupStringByIndex("EntityNames", idx.(int32))
			if !found {
				return nil
			}

			as, err := getCharacter(name)
			if err != nil {
				t.Error(err)
				return nil
			}

			//log.Println(e, p)
			if firstTick == 0 {
				firstTick = p.Tick
			}

			tc := as.GetTransformControl("rootTransform")
			var posLayer *sfm.LogLayer[vector.Vector3[float32]]
			var rotLayer *sfm.LogLayer[vector.Quaternion[float32]]
			posLayer = any(tc.PositionChannel.Log.GetLayer("vector3 log")).(*sfm.LogLayer[vector.Vector3[float32]])
			rotLayer = any(tc.OrientationChannel.Log.GetLayer("quaternion log")).(*sfm.LogLayer[vector.Quaternion[float32]])

			time := float32(p.Tick-firstTick) / 30.

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

func getCharacter(name string) (*sfm.AnimationSet, error) {
	if c, exist := characters[name]; exist {
		return c, nil
	}

	c, err := dota2.NewCharacter(name)
	if err != nil {
		return nil, err
	}

	as, err := c.CreateGameModel(clip)
	if err != nil {
		return nil, err
	}

	tc := as.GetTransformControl("rootTransform")
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
	characters[name] = as

	return as, nil
}

func writeSession(t *testing.T) {
	err := session.WriteBinaryFile(path.Join(varFolder, "test_session.dmx"))
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
