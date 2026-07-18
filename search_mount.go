package main
import ("fmt";"os";"strings")
func main() {
	d,_:=os.ReadFile("cmd/desktop/web-ui/dist/assets/index-1iBlH2r-.js")
	s:=string(d)
	// Search for bHe definition
	i:=strings.Index(s,"bHe=")
	if i<0 {i=strings.Index(s,"bHe ") }
	fmt.Printf("bHe= at %d\n",i)
	if i>=0 {
		start:=i-10;if start<0{start=0}
		end:=i+30;if end>len(s){end=len(s)}
		fmt.Printf("Context: %s\n",s[start:end])
	}
	// Search for c7t definition
	j:=strings.Index(s,"var c7t=")
	if j<0 {j=strings.Index(s,"c7t=") }
	fmt.Printf("c7t= at %d\n",j)
	// Search for SHe definition (createPinia)
	k:=strings.Index(s,"SHe=")
	fmt.Printf("SHe= at %d\n",k)
	// Search for $Xe definition (router)
	l:=strings.Index(s,"$Xe=")
	fmt.Printf("$Xe= at %d\n",l)
	if l>=0 {
		start:=l-20;if start<0{start=0}
		end:=l+40;if end>len(s){end=len(s)}
		fmt.Printf("$Xe context: %s\n",s[start:end])
	}
}
