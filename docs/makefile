.PHONY: docs clean hugo-mod docs-local docs-drafts netlify

docs: 
	hugo server --disableFastRender

clean:
	rm -rf ./public

hugo-mod:
	hugo mod clean
	hugo mod get
	hugo mod vendor

docs-local: clean 
	hugo

docs-drafts:
	hugo server -D --disableFastRender