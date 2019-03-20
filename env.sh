export GOPATH="$PWD"

go get github.com/lunixbochs/struc
go get github.com/docopt/docopt-go
go get golang.org/x/sync/semaphore

rmdata() {
	if [ "$1" != "-f" ]; then
		echo -n '"sudo find ./data -type -f -delete" Are you sure? (y/N) '
		read REPLY
		if [ "$REPLY" != "y" ]; then
			echo "Abort."
			return
		fi
		echo
	fi
	sudo find ./data -type f -delete
}
export rmdata

chkdata() {
	find ./data -type f | xargs -r openssl sha1 | sort
}
export chkdata
