import sys  
sys.stdout.reconfigure(encoding='utf-8')  
with open(r'F:\syproject\goui\pkg\widget\overlay.go', 'r', encoding='utf-8') as f:  
    content = f.read()  
    f.close()  
content = content.rstrip()  
overlay_code = '''  
  
// HitTestOverlays 仅测试浮层 Element 树，忽略骨架。  
func (e *OverlayHostElement) HitTestOverlays(x, y float64) Element {  
	for i := len(e.overlays) - 1; i ; i-- {  
		if el := hitTestOverlayTree(e.overlays[i].el, x, y); el != nil {  
			return el  
		}  
	}  
	return nil  
}  
