K3C = "k3c"

all:
	${K3C} build --target bin -o ./ .

clean:
	rm -rf bin
