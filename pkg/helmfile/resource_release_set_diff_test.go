package helmfile

import (
	"testing"
)

// mockDiffChecker implements diffChecker for unit testing markDiffOutputs.
type mockDiffChecker struct {
	changes     map[string]bool // keys that have changes
	newComputed map[string]bool // keys marked as computed via SetNewComputed
}

func newMockDiffChecker(changedKeys ...string) *mockDiffChecker {
	m := &mockDiffChecker{
		changes:     make(map[string]bool),
		newComputed: make(map[string]bool),
	}
	for _, k := range changedKeys {
		m.changes[k] = true
	}
	return m
}

func (m *mockDiffChecker) HasChange(key string) bool {
	return m.changes[key]
}

func (m *mockDiffChecker) SetNewComputed(key string) error {
	m.newComputed[key] = true
	return nil
}

func TestMarkDiffOutputs_InputChanges_MarksBothComputed(t *testing.T) {
	// When an input attribute has changed, both diff_output and apply_output
	// must be marked as computed to avoid "inconsistent final plan" errors
	// during apply's plan expansion.
	inputKeys := []string{KeyValues, KeyContent, KeyKubeconfig}

	for _, changedKey := range inputKeys {
		t.Run("changed_"+changedKey, func(t *testing.T) {
			d := newMockDiffChecker(changedKey)

			// diff is empty (no changes detected during plan), but input changed
			markDiffOutputs(d, "", inputKeys)

			if !d.newComputed[KeyDiffOutput] {
				t.Errorf("expected diff_output to be marked computed when %s changed", changedKey)
			}
			if !d.newComputed[KeyApplyOutput] {
				t.Errorf("expected apply_output to be marked computed when %s changed", changedKey)
			}
		})
	}
}

func TestMarkDiffOutputs_InputChangesWithDiff_MarksBothComputed(t *testing.T) {
	// Even when helmfile diff found changes, if inputs also changed,
	// we mark both as computed (the diff content may change during apply).
	d := newMockDiffChecker(KeyValues)
	inputKeys := []string{KeyValues, KeyContent}

	markDiffOutputs(d, "some diff output", inputKeys)

	if !d.newComputed[KeyDiffOutput] {
		t.Error("expected diff_output to be marked computed when inputs changed, even with diff")
	}
	if !d.newComputed[KeyApplyOutput] {
		t.Error("expected apply_output to be marked computed when inputs changed, even with diff")
	}
}

func TestMarkDiffOutputs_NoInputChanges_DiffPresent_MarksOnlyApplyOutput(t *testing.T) {
	// When no inputs changed but helmfile diff found changes, only apply_output
	// needs to be computed (diff_output is stable and already set by DiffReleaseSet).
	d := newMockDiffChecker() // no changes
	inputKeys := []string{KeyValues, KeyContent}

	markDiffOutputs(d, "some diff output", inputKeys)

	if d.newComputed[KeyDiffOutput] {
		t.Error("expected diff_output to NOT be marked computed when no inputs changed")
	}
	if !d.newComputed[KeyApplyOutput] {
		t.Error("expected apply_output to be marked computed when diff is present")
	}
}

func TestMarkDiffOutputs_NoInputChanges_NoDiff_MarksNothing(t *testing.T) {
	// When nothing changed and no diff, no outputs should be marked computed.
	// This is the steady-state "no changes needed" case.
	d := newMockDiffChecker() // no changes
	inputKeys := []string{KeyValues, KeyContent}

	markDiffOutputs(d, "", inputKeys)

	if d.newComputed[KeyDiffOutput] {
		t.Error("expected diff_output to NOT be marked computed when nothing changed")
	}
	if d.newComputed[KeyApplyOutput] {
		t.Error("expected apply_output to NOT be marked computed when nothing changed")
	}
}

func TestMarkDiffOutputs_MultipleInputChanges(t *testing.T) {
	// Multiple inputs changed — should still mark both computed.
	d := newMockDiffChecker(KeyValues, KeyContent, KeyKubeconfig)
	inputKeys := []string{KeyValues, KeyContent, KeyKubeconfig}

	markDiffOutputs(d, "", inputKeys)

	if !d.newComputed[KeyDiffOutput] {
		t.Error("expected diff_output to be marked computed")
	}
	if !d.newComputed[KeyApplyOutput] {
		t.Error("expected apply_output to be marked computed")
	}
}

func TestMarkDiffOutputs_IrrelevantKeyChanged(t *testing.T) {
	// A key changed that is NOT in the inputKeys list — should not trigger computed.
	d := newMockDiffChecker("some_other_key")
	inputKeys := []string{KeyValues, KeyContent}

	markDiffOutputs(d, "", inputKeys)

	if d.newComputed[KeyDiffOutput] {
		t.Error("expected diff_output to NOT be marked computed for irrelevant key change")
	}
	if d.newComputed[KeyApplyOutput] {
		t.Error("expected apply_output to NOT be marked computed for irrelevant key change")
	}
}

func TestMarkDiffOutputs_ReleaseSetInputKeys(t *testing.T) {
	// Verify that the release set input keys used in resourceReleaseSetDiff
	// are all recognized — changing any of them marks outputs computed.
	releaseSetInputKeys := []string{
		KeyValues, KeyValuesFiles, KeyContent, KeyPath, KeyWorkingDirectory,
		KeyEnvironment, KeyEnvironmentVariables, KeyBin, KeyHelmBin,
		KeySelector, KeySelectors, KeyKubeconfig,
	}

	for _, key := range releaseSetInputKeys {
		t.Run(key, func(t *testing.T) {
			d := newMockDiffChecker(key)
			markDiffOutputs(d, "", releaseSetInputKeys)

			if !d.newComputed[KeyDiffOutput] {
				t.Errorf("expected diff_output to be marked computed when %s changed", key)
			}
			if !d.newComputed[KeyApplyOutput] {
				t.Errorf("expected apply_output to be marked computed when %s changed", key)
			}
		})
	}
}

func TestMarkDiffOutputs_ReleaseInputKeys(t *testing.T) {
	// Verify that the release input keys used in resourceHelmfileReleaseDiff
	// are all recognized — changing any of them marks outputs computed.
	releaseInputKeys := []string{
		KeyValues, KeyChart, KeyVersion, KeyWorkingDirectory,
		KeyKubeconfig, KeyKubecontext, KeyBin, KeyHelmBin,
		KeyNamespace, KeyName,
	}

	for _, key := range releaseInputKeys {
		t.Run(key, func(t *testing.T) {
			d := newMockDiffChecker(key)
			markDiffOutputs(d, "", releaseInputKeys)

			if !d.newComputed[KeyDiffOutput] {
				t.Errorf("expected diff_output to be marked computed when %s changed", key)
			}
			if !d.newComputed[KeyApplyOutput] {
				t.Errorf("expected apply_output to be marked computed when %s changed", key)
			}
		})
	}
}
