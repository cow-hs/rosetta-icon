/*
 * Copyright 2020 ICON Foundation
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package state

import (
	"bytes"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
)

type objectGraph struct {
	bk        db.Bucket
	nextHash  int
	graphHash []byte
	graphData []byte
}

func (o *objectGraph) flush() error {
	if o.bk == nil || o.graphData == nil {
		return nil
	}
	prevData, err := o.bk.Get(o.graphHash)
	if err != nil {
		return err
	}
	// already exists
	if prevData != nil {
		return nil
	}
	if err := o.bk.Set(o.graphHash, o.graphData); err != nil {
		return err
	}
	return nil
}

func (o *objectGraph) Equal(o2 *objectGraph) bool {
	if o == o2 {
		return true
	}
	if o == nil || o2 == nil {
		return false
	}
	if o.nextHash != o2.nextHash {
		return false
	}
	if !bytes.Equal(o.graphHash, o2.graphHash) {
		return false
	}
	return true
}

func (o *objectGraph) RLPEncodeSelf(e codec.Encoder) error {
	return e.EncodeListOf(o.nextHash, o.graphHash)
}

func (o *objectGraph) RLPDecodeSelf(d codec.Decoder) error {
	return d.DecodeListOf(&o.nextHash, &o.graphHash)
}

func (o *objectGraph) Changed(
	dbase db.Database, hasData bool, nextHash int, graphData []byte,
) (*objectGraph, error) {
	n := new(objectGraph)
	if o != nil {
		*n = *o
	} else {
		if bk, err := dbase.GetBucket(db.BytesByHash); err != nil {
			return nil, errors.CriticalIOError.Wrap(err, "FailToGetBucket")
		} else {
			n.bk = bk
		}
	}
	n.nextHash = nextHash
	if hasData {
		if len(graphData) == 0 {
			n.graphData = nil
			n.graphHash = nil
		} else {
			n.graphData = graphData
			n.graphHash = crypto.SHA3Sum256(graphData)
		}
	}
	if n.nextHash == 0 && len(n.graphHash) == 0 {
		return nil, nil
	}
	return n, nil
}

func (o *objectGraph) Get(withData bool) (int, []byte, []byte, error) {
	if o == nil {
		return 0, nil, nil, errors.ErrNotFound
	}
	if withData {
		if o.graphData == nil && len(o.graphHash) > 0 {
			v, err := o.bk.Get(o.graphHash)
			if err != nil {
				err = errors.CriticalIOError.Wrap(err, "FailToGetValue")
				return 0, nil, nil, err
			}
			if v == nil {
				err = errors.NotFoundError.Errorf(
					"NoValueInHash(hash=%#x)", o.graphHash)
				return 0, nil, nil, err
			}
			o.graphData = v
		}
		return o.nextHash, o.graphHash, o.graphData, nil
	} else {
		return o.nextHash, o.graphHash, nil, nil
	}
}
