all:
	@echo "Targets: "
	@make -qpRr | egrep -e '^[a-z].*:$$' | sed -e 's~:~~g' | grep -v 'all' | sort
pull:
	git checkout master
	git pull
commit:
	test -z "$$(git status --short)" || opencode run 'commit it'
build:
	bash build.sh
