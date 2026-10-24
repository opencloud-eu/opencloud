# Tools

PNPM ?= pnpm
MSGCAT ?= msgcat
MSGMERGE ?= msgmerge
MSGFMT ?= msgfmt

# Variables

POT = translation.pot
POS ?= $(wildcard *.po)

# Translations

.PHONY: build
build: json

.PHONY: json
json:
	@for po in $(POS); do \
		lang=$$(echo $$po | sed "s/\.po//"); \
		$(PNPM) i18next-conv -K --skipUntranslated -l $$lang -s $$po -t ../src/locales/$$lang/translation.json; \
	done
	$(PNPM) node build-json.js ../src/locales/locales.json $(POS)

.PHONY: extract-keys
extract-keys:
	$(PNPM) i18next-cli extract -q || true

translation.en.json: FORCE
	$(MAKE) extract-keys

.PHONY: extract
extract: pot

.PHONY: pot
pot: translation.en.json
	@tmpfile1=$(shell mktemp).po; \
	tmpfile2=$(shell mktemp).po; \
	trap 'rm -f "$$tmpfile1" "$$tmpfile2"' EXIT; \
	$(PNPM) i18next-conv --project "LibreGraph Connect Identifier" -K -l en -s translation.en.json -t $$tmpfile1; \
	$(PNPM) node build-pot.js $$tmpfile1 $$tmpfile2; \
	$(MSGCAT) --no-wrap -o $(POT) $$tmpfile2

.PHONY: merge
merge: $(POS)

$(POS): FORCE $(POT)
	@echo -n "$@ " && \
	$(MSGMERGE) -U \
		--backup=none \
		--no-wrap \
		--sort-output \
		$@ $(POT)

.PHONY: stats
stats:
	$(foreach po, $(POS), $(shell $(MSGFMT) -v --statistics $(po)))
	@- true

FORCE:
