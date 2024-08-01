package main

import (
	"errors"
	"strconv"
)

type Replay struct {
	playersByHandle    map[uint64]uint64
	playersByAccountId map[uint64]uint64
	itemsPerAccount    map[uint64]map[uint32]struct{}
	units              map[uint64]uint64
}

func NewReplay() *Replay {
	return &Replay{
		playersByHandle:    make(map[uint64]uint64),
		playersByAccountId: make(map[uint64]uint64),
		itemsPerAccount:    make(map[uint64]map[uint32]struct{}),
		units:              make(map[uint64]uint64),
	}
}

func (r *Replay) AddWearable(attributes map[string]any) error {
	var value any
	var ok bool
	var itemDefinitionIndex uint32
	var accountID uint64

	if value, ok = attributes["m_iItemDefinitionIndex"]; !ok {
		return errors.New("can find m_iItemDefinitionIndex")
	}

	if itemDefinitionIndex, ok = value.(uint32); !ok {
		return errors.New("wrong type for m_iItemDefinitionIndex")
	}

	if value, ok = attributes["m_iAccountID"]; !ok {
		return errors.New("can find m_iAccountID")
	}

	if accountID, ok = value.(uint64); !ok {
		return errors.New("wrong type for m_iAccountID")
	}

	if _, ok = r.itemsPerAccount[accountID]; !ok {
		r.itemsPerAccount[accountID] = map[uint32]struct{}{}
	}
	r.itemsPerAccount[accountID][itemDefinitionIndex] = struct{}{}

	return nil
}

func (r *Replay) AddPlayerController(attributes map[string]any, handle uint64) error {
	var value any
	var ok bool
	var SteamID uint64

	if value, ok = attributes["m_steamID"]; !ok {
		return errors.New("can find m_steamID")
	}

	if SteamID, ok = value.(uint64); !ok {
		return errors.New("wrong type for m_steamID")
	}

	accountId := SteamID - BASE_STEAM_ID
	r.playersByHandle[handle] = accountId
	r.playersByAccountId[accountId] = handle

	return nil
}

func (r *Replay) AddUnit(attributes map[string]any, handle uint64) error {
	var value any
	var ok bool
	var ownerEntity uint64

	if value, ok = attributes["m_hOwnerEntity"]; !ok {
		return errors.New("can find m_hOwnerEntity")
	}

	if ownerEntity, ok = value.(uint64); !ok {
		return errors.New("wrong type for m_hOwnerEntity")
	}
	r.units[handle] = ownerEntity

	return nil
}

func (r *Replay) GetItems(handle uint64) []string {
	i := make([]string, 0, 5)

	var ownerId uint64
	var accountId uint64
	var items map[uint32]struct{}
	var ok bool

	if ownerId, ok = r.units[handle]; !ok {
		return i
	}

	if accountId, ok = r.playersByHandle[ownerId]; !ok {
		return i

	}
	if items, ok = r.itemsPerAccount[accountId]; !ok {
		return i
	}

	for id := range items {
		i = append(i, strconv.Itoa(int(id)))
	}

	return i
}
