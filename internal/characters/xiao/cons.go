package xiao

import (
	"github.com/genshinsim/gcsim/pkg/core/action"
	"github.com/genshinsim/gcsim/pkg/core/attributes"
	"github.com/genshinsim/gcsim/pkg/core/combat"
	"github.com/genshinsim/gcsim/pkg/core/event"
	"github.com/genshinsim/gcsim/pkg/core/glog"
)

// Implements Xiao C2:
// When in the party and not on the field, Xiao's Energy Recharge is increased by 25%
func (c *char) c2() {
	m := make([]float64, attributes.EndStatType)
	m[attributes.ER] = 0.25
	c.AddStatMod("xiao-c2", -1, attributes.ER, func() ([]float64, bool) {
		if c.Core.Player.Active() != c.Index {
			return m, true
		}
		return nil, false
	})
}

// Implements Xiao C6:
// While under the effect of Bane of All Evil, hitting at least 2 opponents with Xiao's Plunge Attack will immediately grant him 1 charge of Lemniscatic Wind Cycling, and for the next 1s, he may use Lemniscatic Wind Cycling while ignoring its CD.
// Adds an OnDamage event checker - if we record two or more instances of plunge damage, then activate C6
func (c *char) c6() {
	c.Core.Events.Subscribe(event.OnDamage, func(args ...interface{}) bool {
		atk := args[1].(*combat.AttackEvent)
		if atk.Info.ActorIndex != c.Index {
			return false
		}
		if !((atk.Info.Abil == "High Plunge") || (atk.Info.Abil == "Low Plunge")) {
			return false
		}
		if c.Core.Status.Duration("xiaoburst") == 0 {
			return false
		}
		// Stops after reaching 2 hits on a single plunge.
		// Plunge frames are greater than duration of C6 so this will always refresh properly.
		if c.Core.Status.Duration("xiaoc6") > 0 {
			return false
		}
		if c.c6Src != atk.SourceFrame {
			c.c6Src = atk.SourceFrame
			c.c6Count = 0
			return false
		}

		c.c6Count++

		// Prevents activation more than once in a single plunge attack
		if c.c6Count == 2 {
			c.ResetActionCooldown(action.ActionSkill)

			c.Core.Status.Add("xiaoc6", 60)
			c.Core.Log.NewEvent("Xiao C6 activated", glog.LogCharacterEvent, c.Index).
				Write("new E charges", c.Tags["eCharge"]).
				Write("expiry", c.Core.F+60)

			c.c6Count = 0
			return false
		}
		return false
	}, "xiao-c6")
}
