// Copyright 2020 ICON Foundation, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package icon

import "time"

const (
	// Blockchain is ICON.
	Blockchain string = "ICON"

	// MainnetNetwork is the value of the network
	// in MainnetNetworkIdentifier.
	MainnetNetwork string = "Mainnet"

	// TestnetNetwork is the value of the network
	// in TestnetNetworkIdentifier.
	TestnetNetwork string = "Testnet"

	// Symbol is the symbol value
	// used in Currency.
	Symbol = "ICX"

	// Decimals is the decimals value
	// used in Currency.
	Decimals = 18

	defaultTimeout = 60 * time.Second
	dialTimeout    = 5 * time.Second

	genesisBlockIndex          = 0
	HistoricalBalanceSupported = false
)

type requestMethod string

const (
	requestMethodGetBlock requestMethod = "icx_getBlock"
)

const (
	requestID      = 1
	jSONRPCVersion = "2.0"
)
