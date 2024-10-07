/*
 * Copyright © 2024 Kaleido, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with
 * the License. You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on
 * an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the
 * specific language governing permissions and limitations under the License.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package smt

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hyperledger-labs/zeto/go-sdk/pkg/sparse-merkle-tree/core"
	"github.com/hyperledger-labs/zeto/go-sdk/pkg/sparse-merkle-tree/node"
	"github.com/kaleido-io/paladin/toolkit/pkg/plugintk"
	"github.com/kaleido-io/paladin/toolkit/pkg/prototk"
	"github.com/kaleido-io/paladin/toolkit/pkg/query"
	"github.com/kaleido-io/paladin/toolkit/pkg/tktypes"
	"gorm.io/gorm"
)

type StatesStorage interface {
	core.Storage
	GetNewStates() []*prototk.NewLocalState
}

type statesStorage struct {
	CoreInterface     plugintk.DomainCallbacks
	smtName           string
	stateQueryContext string
	rootSchemaId      string
	nodeSchemaId      string
	rootNode          core.NodeIndex
	newNodes          []*prototk.NewLocalState
}

func NewStatesStorage(c plugintk.DomainCallbacks, smtName, stateQueryContext, rootSchemaId, nodeSchemaId string) StatesStorage {
	return &statesStorage{
		CoreInterface:     c,
		smtName:           smtName,
		stateQueryContext: stateQueryContext,
		rootSchemaId:      rootSchemaId,
		nodeSchemaId:      nodeSchemaId,
	}
}

func (s *statesStorage) GetNewStates() []*prototk.NewLocalState {
	return s.newNodes
}

func (s *statesStorage) GetRootNodeIndex() (core.NodeIndex, error) {
	if s.rootNode != nil {
		return s.rootNode, nil
	}
	queryBuilder := query.NewQueryBuilder().
		Limit(1).
		Sort(".created DESC").
		Equal("smtName", s.smtName)

	res, err := s.CoreInterface.FindAvailableStates(context.Background(), &prototk.FindAvailableStatesRequest{
		StateQueryContext: s.stateQueryContext,
		SchemaId:          s.rootSchemaId,
		QueryJson:         queryBuilder.Query().String(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to find available states. %s", err)
	}

	if len(res.States) == 0 {
		return nil, core.ErrNotFound
	}

	var root MerkleTreeRoot
	err = json.Unmarshal([]byte(res.States[0].DataJson), &root)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal root node index. %s", err)
	}

	idx, err := node.NewNodeIndexFromHex(root.RootIndex)
	return idx, err
}

func (s *statesStorage) UpsertRootNodeIndex(root core.NodeIndex) error {
	newRoot := &MerkleTreeRoot{
		SmtName:   s.smtName,
		RootIndex: "0x" + root.Hex(),
	}
	data, err := json.Marshal(newRoot)
	if err != nil {
		return fmt.Errorf("failed to upsert root node. %s", err)
	}
	newRootState := &prototk.NewLocalState{
		Id:            &newRoot.RootIndex,
		SchemaId:      s.rootSchemaId,
		StateDataJson: string(data),
		PreConfirmed:  true, // merkle tree states are local and written immediately
	}
	s.newNodes = append(s.newNodes, newRootState)
	s.rootNode = root
	return err
}

func (s *statesStorage) GetNode(ref core.NodeIndex) (core.Node, error) {
	// the node's reference key (not the index) is used as the key to
	// store the node in the DB
	queryBuilder := query.NewQueryBuilder().
		Limit(1).
		Sort(".created").
		Equal("refKey", ref.Hex())

	res, err := s.CoreInterface.FindAvailableStates(context.Background(), &prototk.FindAvailableStatesRequest{
		StateQueryContext: s.stateQueryContext,
		SchemaId:          s.nodeSchemaId,
		QueryJson:         queryBuilder.Query().String(),
	})
	if err == gorm.ErrRecordNotFound {
		return nil, core.ErrNotFound
	} else if err != nil {
		return nil, fmt.Errorf("failed to find available states. %s", err)
	}
	if len(res.States) == 0 {
		return nil, core.ErrNotFound
	}
	var n MerkleTreeNode
	err = json.Unmarshal([]byte(res.States[0].DataJson), &n)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal Merkle Tree Node from state json. %s", err)
	}

	var newNode core.Node
	nodeType := core.NodeTypeFromByte(n.Type[:][0])
	switch nodeType {
	case core.NodeTypeLeaf:
		idx, err1 := node.NewNodeIndexFromHex(n.Index.HexString())
		if err1 != nil {
			return nil, fmt.Errorf("failed to create lead node index. %s", err1)
		}
		v := node.NewIndexOnly(idx)
		newNode, err = node.NewLeafNode(v)
	case core.NodeTypeBranch:
		leftChild, err1 := node.NewNodeIndexFromHex(n.LeftChild.HexString())
		if err1 != nil {
			return nil, fmt.Errorf("failed to create left child node index. %s", err1)
		}
		rightChild, err2 := node.NewNodeIndexFromHex(n.RightChild.HexString())
		if err2 != nil {
			return nil, fmt.Errorf("failed to create right child node index. %s", err2)
		}
		newNode, err = node.NewBranchNode(leftChild, rightChild)
	}
	return newNode, err
}

func (s *statesStorage) InsertNode(n core.Node) error {
	// we clone the node so that the value properties are not saved
	refKeyBytes, err := tktypes.ParseBytes32(n.Ref().Hex())
	if err != nil {
		return err
	}
	newNode := &MerkleTreeNode{
		RefKey: refKeyBytes,
		Type:   tktypes.HexBytes([]byte{n.Type().ToByte()}),
	}
	if n.Type() == core.NodeTypeBranch {
		left := n.LeftChild().Hex()
		leftBytes, err := tktypes.ParseBytes32(left)
		if err != nil {
			return err
		}
		newNode.LeftChild = leftBytes
		right := n.RightChild().Hex()
		rightBytes, err := tktypes.ParseBytes32(right)
		if err != nil {
			return err
		}
		newNode.RightChild = rightBytes
	} else if n.Type() == core.NodeTypeLeaf {
		idx := n.Index().Hex()
		idxBytes, err := tktypes.ParseBytes32(idx)
		if err != nil {
			return err
		}
		newNode.Index = idxBytes
	}

	data, err := json.Marshal(newNode)
	if err != nil {
		return fmt.Errorf("failed to insert node. %s", err)
	}
	refKey := newNode.RefKey.HexString()
	newNodeState := &prototk.NewLocalState{
		Id:            &refKey,
		SchemaId:      s.nodeSchemaId,
		StateDataJson: string(data),
	}
	s.newNodes = append(s.newNodes, newNodeState)
	return err
}

func (s *statesStorage) BeginTx() (core.Transaction, error) {
	// not needed for this implementation because the DB transaction
	// is already enforced by the core interface
	return s, nil
}

func (s *statesStorage) Commit() error {
	// not needed for this implementation because the DB transaction
	// is already enforced by the core interface
	return nil
}

func (s *statesStorage) Rollback() error {
	// not needed for this implementation because the DB transaction
	// is already enforced by the core interface
	return nil
}

func (s *statesStorage) Close() {
	// not needed for this implementation because
	// there are no resources to close
}
