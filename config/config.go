//
// Copyright (C) 2022 YumeMichi
//
// SPDX-License-Identifier: Apache-2.0
//

package config

var Conf = &Config{}

func init() {
	Conf = Load("./config.yml")
}
