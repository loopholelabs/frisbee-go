/*
	Copyright 2022 Loophole Labs

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

// Package conn contains functionality for helping with
// testing networking code that requires creating and tearing
// down net.Conn connections
package conn

// Network specifies that TCP connections will be created
const Network = "tcp"

// Listen specifies that the listener will bind to localhost on a random port
const Listen = "127.0.0.1:0"
