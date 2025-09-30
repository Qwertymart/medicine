# -*- coding: utf-8 -*-
import os
import json
from typing import Any, Dict, Optional

from fastapi import FastAPI, Body, Query, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import JSONResponse

# ВАЖНО: файл serve_inference.py должен лежать рядом (мы писали его ранее)
from serve_inference import InferenceService

APP_TITLE = "Itelma ML Inference API"
APP_VERSION = "1.0.0"

OUT_DIR = os.getenv("OUT_DIR", os.path.abspath(os.path.join(os.getcwd(), "out")))

app = FastAPI(title=APP_TITLE, version=APP_VERSION)

# CORS — при необходимости сузьте источники
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["POST", "GET", "OPTIONS"],
    allow_headers=["*"],
)

# Глобальный синглтон сервиса
SVC: Optional[InferenceService] = None

def summarize(resp: Dict[str, Any]) -> Dict[str, Any]:
    """
    Готовит компактные строки под UI: тренд, риски, сводку на 60 мин.
    """
    out = {}
    if not resp.get("ok"):
        return out

    res = resp.get("result", {})

    # тренд на русском языке
    if "trend5_trend" in res:
        tr = res["trend5_trend"]
        if isinstance(tr, dict) and "proba" in tr:
            classes = list(tr["proba"].keys())
            probs   = list(tr["proba"].values())
            if classes and probs:
                best_i  = int(max(range(len(probs)), key=lambda i: probs[i]))
                
                # Перевод классов тренда на русский
                class_names = {
                    "down": "снижение",
                    "flat": "стабильное",
                    "up": "повышение"
                }
                
                # Уровень уверенности
                confidence_level = "высокая" if probs[best_i] > 0.7 else "средняя" if probs[best_i] > 0.5 else "низкая"
                
                russian_class = class_names.get(classes[best_i], classes[best_i])
                out["trend_text"] = f"Тенденция изменения показателей (5 мин): {russian_class}, уверенность {confidence_level} ({probs[best_i]*100:.0f}%)"
                out["trend_probs"] = {class_names.get(k, k): round(v*100.0, 1) for k, v in tr["proba"].items()}

    # риски по горизонтам
    for h in ["h15","h30","h45","h60"]:
        if h in res:
            p = res[h].get("proba", None)
            if p is not None:
                out[h] = {
                    "risk_pct": round(float(p)*100.0, 2),
                    "pred": int(res[h].get("pred", 0)),
                    "thr": res[h].get("thr"),
                }

    # медицинское заключение на русском языке
    if "h15" in out and "h30" in out and "h45" in out and "h60" in out:
        h15_risk = out["h15"]["risk_pct"]
        h30_risk = out["h30"]["risk_pct"] 
        h45_risk = out["h45"]["risk_pct"]
        h60_risk = out["h60"]["risk_pct"]
        
        # Формирование прогнозов по времени
        forecasts = []
        forecasts.append(f"Вероятность развития гипоксии плода в следующие 15 минут: {h15_risk:.1f}%")
        forecasts.append(f"Вероятность развития гипоксии плода в следующие 30 минут: {h30_risk:.1f}%")
        forecasts.append(f"Вероятность развития гипоксии плода в следующие 45 минут: {h45_risk:.1f}%")
        forecasts.append(f"Вероятность развития гипоксии плода в следующие 60 минут: {h60_risk:.1f}%")
        
        # Определение общего уровня риска
        max_risk = max(h15_risk, h30_risk, h45_risk, h60_risk)
        
        # Медицинские рекомендации
        if max_risk >= 20:
            clinical_decision = "ТРЕБУЕТСЯ СРОЧНОЕ ВМЕШАТЕЛЬСТВО! Высокий риск развития осложнений."
            risk_category = "критический"
        elif max_risk >= 10:
            clinical_decision = "Рекомендуется усиленное наблюдение. Умеренный риск осложнений."
            risk_category = "повышенный"
        elif max_risk >= 5:
            clinical_decision = "Показано продолжение мониторинга. Низкий риск осложнений."
            risk_category = "низкий"
        else:
            clinical_decision = "Показатели в норме. Состояние плода стабильное."
            risk_category = "минимальный"
        
        # Составляем итоговое заключение
        prediction_text = "ПРЕДИКТИВНЫЙ АНАЛИЗ СОСТОЯНИЯ ПЛОДА\n\n"
        prediction_text += "Краткосрочные прогнозы:\n"
        prediction_text += "\n".join(f"• {forecast}" for forecast in forecasts)
        prediction_text += f"\n\nОбщий уровень риска: {risk_category}\n"
        prediction_text += f"Клиническое заключение: {clinical_decision}"
        
        out["summary"] = {
            "risk_60m": h60_risk,
            "ok_60m": round(100.0 - h60_risk, 2),
            "text": prediction_text,
            "clinical_decision": clinical_decision,
            "risk_category": risk_category,
            "forecasts": forecasts
        }
    
    return out

@app.on_event("startup")
def _on_startup():
    global SVC
    try:
        SVC = InferenceService(out_dir=OUT_DIR)
        print(f"[startup] InferenceService loaded. OUT_DIR={OUT_DIR}")
        # перечислим, что загрузилось
        specs = SVC.hub.list_specs()
        print("[startup] models loaded:", list(specs.keys()))
        for name, spec in specs.items():
            if not spec.features:
                print(f"[startup] ⚠ meta not found or empty for {name}")
    except Exception as e:
        # если сервис не поднялся — пусть API честно вернёт 500 позже
        print("[startup] ERROR:", repr(e))
        SVC = None

@app.get("/health")
def health():
    ok = SVC is not None
    return {
        "ok": ok,
        "out_dir": OUT_DIR,
        "models": list(SVC.hub.list_specs().keys()) if ok else [],
        "msg": "ready" if ok else "service not initialized"
    }

@app.get("/meta")
def meta():
    if SVC is None:
        raise HTTPException(status_code=500, detail="service not initialized")
    info = {}
    for name, spec in SVC.hub.list_specs().items():
        info[name] = {
            "kind": spec.kind,
            "features_count": len(spec.features),
            "required_window_sec": spec.required_window_sec,
            "has_threshold": spec.threshold is not None,
            "classes": spec.classes,
        }
    return {"ok": True, "out_dir": OUT_DIR, "meta": info}

@app.post("/infer")
def infer(
    payload: Dict[str, Any] = Body(..., description="One-tick feature row from backend"),
    verbose: bool = Query(False, description="Add UI-friendly summary fields"),
):
    if SVC is None:
        raise HTTPException(status_code=500, detail="service not initialized")
    try:
        resp = SVC.handle(payload)
        if verbose:
            resp["ui"] = summarize(resp)
        return JSONResponse(resp)
    except Exception as e:
        raise HTTPException(status_code=400, detail=f"bad request: {type(e).__name__}: {e}")

# удобный запуск:  uvicorn app:app --host 0.0.0.0 --port 8000 --workers 1
if __name__ == "__main__":
    import uvicorn
    uvicorn.run("app:app", host="0.0.0.0", port=int(os.getenv("PORT", "8000")), reload=False)
