package baizhu

import (
	"github.com/genshinsim/gcsim/pkg/core/attributes"
	"github.com/genshinsim/gcsim/pkg/core/player/character"
	"github.com/genshinsim/gcsim/pkg/modifier"
)

const a1BuffKey = "baizhu-a1"

func (c *char) a1() {
	m := make([]float64, attributes.EndStatType)
	m[attributes.DendroP] = 0.25
	c.AddStatMod(character.StatMod{
		Base:         modifier.NewBase("baizhu-a1", -1),
		AffectedStat: attributes.DendroP,
		Amount: func() ([]float64, bool) {
			return m, true
		},
	})
}

func (c *char) a4() {
	return
}