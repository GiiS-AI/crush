package giis

import (
	"testing"

	giisskills "github.com/GiiS-AI/GiiS-Code/internal/skills"
	"github.com/stretchr/testify/require"
)

func TestToolsDisabledWithoutEnv(t *testing.T) {
	t.Setenv(serverURLEnv, "")
	t.Setenv(patEnv, "")

	require.Nil(t, Tools())
}

func TestToolsAndBuiltinSkillEnabledWithEnv(t *testing.T) {
	t.Setenv(serverURLEnv, "https://chat.giis.ai")
	t.Setenv(patEnv, "test-token")

	tools := Tools()
	require.Len(t, tools, 3)
	require.Equal(t, scrapeToolName, tools[0].Info().Name)
	require.Equal(t, emailToolName, tools[1].Info().Name)
	require.Equal(t, crmToolName, tools[2].Info().Name)

	skills, _ := giisskills.DiscoverBuiltinWithStates()
	var found bool
	for _, skill := range skills {
		if skill.Name == SkillName {
			found = true
			break
		}
	}
	require.True(t, found)
}
