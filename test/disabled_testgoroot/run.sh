find "$GOROOT/src/pkg" -type d| while read d; do
	find "$d" -name \*_test.go -type f -maxdepth 1|while read f; do
		(cd $d; $GOSLOPPY test) || (echo error while building $d; exit -1)
		break
	done
done
