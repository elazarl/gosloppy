find "$GOROOT/src/pkg" -type d -not -name reflect | grep -vw runtime | grep -v '\(net\|time\|syscall\|sync\|testing\)' | while read d; do
	find "$d" -name \*_test.go -type f -maxdepth 1|while read f; do
		(cd $d; $GOSLOPPY test) || (echo error while building $d; exit -1)
		break
	done
done
