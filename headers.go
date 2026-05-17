package floopy

import "strconv"

// buildFloopyHeaders translates Options into wire headers. A nil Options or
// nil field is omitted. Booleans are rendered lowercase ("true"/"false") to
// match the Node SDK's String(boolean).
func buildFloopyHeaders(o *Options) map[string]string {
	h := map[string]string{}
	if o == nil {
		return h
	}
	if o.Cache != nil {
		if o.Cache.Enabled != nil {
			h[headerCacheEnabled] = boolStr(*o.Cache.Enabled)
		}
		if o.Cache.BucketMaxSize != nil {
			h[headerCacheBucketMaxSize] = strconv.Itoa(*o.Cache.BucketMaxSize)
		}
	}
	if o.PromptID != "" {
		h[headerPromptID] = o.PromptID
	}
	if o.PromptVersion != "" {
		h[headerPromptVersion] = o.PromptVersion
	}
	if o.LLMSecurityEnabled != nil {
		h[headerLLMSecurityEnabled] = boolStr(*o.LLMSecurityEnabled)
	}
	return h
}

// mergeHeaders merges header layers with later layers winning. nil layers are
// skipped.
func mergeHeaders(layers ...map[string]string) map[string]string {
	merged := map[string]string{}
	for _, layer := range layers {
		for k, v := range layer {
			merged[k] = v
		}
	}
	return merged
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
