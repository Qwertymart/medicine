# -*- coding: utf-8 -*-
import os, re, json, math
from typing import Dict, Any, List, Optional, Tuple
import numpy as np
import pandas as pd
from joblib import load

# ---------- утилиты ----------
def _numeric(x) -> bool:
    return isinstance(x, (int, float, np.integer, np.floating)) and not (isinstance(x, float) and (math.isnan(x) or math.isinf(x)))

def _win_sec_from_feat_name(name: str) -> Optional[int]:
    # ожидаем f_{N}s_...
    m = re.match(r"^f_(\d+)s_", name)
    return int(m.group(1)) if m else None

# ---------- загрузка мета и моделей ----------
class ModelSpec:
    def __init__(self, name: str, features: List[str], kind: str, threshold: Optional[float] = None, classes: Optional[List[str]] = None):
        self.name = name         # 'trend5_trend' | 'h15' | ...
        self.features = features # список имён фич (из meta)
        self.kind = kind         # 'trend' или 'risk'
        self.threshold = threshold
        self.classes = classes or []

    @property
    def required_window_sec(self) -> int:
        # по фичам модели определяем максимальное окно
        wins = [w for w in (_win_sec_from_feat_name(c) for c in self.features) if w]
        return max(wins) if wins else 0

class ModelHub:
    """
    Простой загрузчик/инференс без состояний.
    """
    def __init__(self, out_dir: str):
        self.out_dir = out_dir
        self.specs: Dict[str, ModelSpec] = {}
        self.models: Dict[str, Any] = {}
        self._load_all()

    def _load_one(self, base: str) -> Optional[ModelSpec]:
        meta_path = os.path.join(self.out_dir, f"{base}__meta.json")
        mdl_path  = os.path.join(self.out_dir, f"{base}.joblib")
        if not (os.path.exists(meta_path) and os.path.exists(mdl_path)):
            return None
        with open(meta_path, "r", encoding="utf-8") as f:
            meta = json.load(f)
        features = meta.get("features", [])
        threshold = meta.get("threshold", None)
        classes = meta.get("classes", None)
        # тип модели
        if base.startswith("h"):
            kind = "risk"
        elif base.startswith("trend"):
            kind = "trend"
        else:
            # по наличию classes считаем это трендом; иначе риск
            kind = "trend" if classes else "risk"
        # загрузка модели
        clf = load(mdl_path)
        self.models[base] = clf
        spec = ModelSpec(base, features, kind, threshold=threshold, classes=classes)
        self.specs[base] = spec
        return spec

    def _load_all(self):
        # жёсткий список имён, которые мы ждём
        for base in ["trend5_trend", "h15", "h30", "h45", "h60"]:
            self._load_one(base)

    def list_specs(self) -> Dict[str, ModelSpec]:
        return self.specs

# ---------- валидация и запуск ----------
class InferenceService:
    """
    Основной сервис: валидирует вход и запускает доступные модели.
    """
    def __init__(self, out_dir: str):
        self.hub = ModelHub(out_dir)

    def _validate_basic(self, payload: Dict[str, Any]) -> List[str]:
        errs = []
        # ИЗМЕНЕНО: теперь проверяем card_id вместо patient_id
        if "card_id" not in payload:
            errs.append("missing: card_id")
        if "features" not in payload or not isinstance(payload["features"], dict):
            errs.append("missing: features (dict)")
        else:
            # типы значений — числа (int/float/np.*), NaN/inf не допускаем
            bad = [k for k, v in payload["features"].items() if not _numeric(v)]
            if bad:
                errs.append(f"non-numeric features: {bad[:5]}{'...' if len(bad)>5 else ''}")
        # необязательные поля
        if "fs_hz" in payload and not _numeric(payload["fs_hz"]):
            errs.append("fs_hz must be numeric")
        if "available_windows" in payload:
            aw = payload["available_windows"]
            if not isinstance(aw, list) or not all(isinstance(x, str) for x in aw):
                errs.append("available_windows must be list[str]")
        return errs

    def _features_available(self, have: Dict[str, Any], need: List[str]) -> Tuple[bool, List[str]]:
        missing = [c for c in need if c not in have]
        return (len(missing) == 0), missing

    def _window_available(self, payload: Dict[str, Any], req_sec: int) -> bool:
        # если бэк прислал available_windows — используем их; иначе пытаемся угадать по фичам
        aw = payload.get("available_windows")
        if isinstance(aw, list) and aw:
            # примеры: ["240s","600s"]
            need_tag = f"{req_sec}s"
            return any(tag == need_tag for tag in aw)
        # fallback: проверяем, есть ли хоть одна фича с нужным префиксом
        have = payload["features"]
        return any(_win_sec_from_feat_name(k) == req_sec for k in have.keys())

    def _runnable_models(self, payload: Dict[str, Any]) -> Dict[str, Dict[str, Any]]:
        """
        Возвращает словарь по моделям: {'status': 'ok'|'skip', 'reason':..., 'missing':[...] }
        """
        have = payload["features"]
        out = {}
        for name, spec in self.hub.list_specs().items():
            # тренд может отсутствовать, если не обучен
            if spec is None or not spec.features:
                out[name] = {"status": "skip", "reason": "model or meta not loaded", "missing": []}
                continue
            # проверка окна
            if spec.required_window_sec and not self._window_available(payload, spec.required_window_sec):
                out[name] = {"status": "skip", "reason": f"need ≥{spec.required_window_sec//60}min window", "missing": []}
                continue
            # проверка наличия фич (по именам из meta)
            ok, missing = self._features_available(have, spec.features)
            if not ok:
                out[name] = {"status": "skip", "reason": "missing features", "missing": missing}
                continue
            out[name] = {"status": "ok", "reason": "", "missing": []}
        return out

    def _run_one(self, name: str, spec: ModelSpec, row_df: pd.DataFrame) -> Dict[str, Any]:
        clf = self.hub.models[name]
        # ВАЖНО: подать в том порядке колонок, как в meta['features']
        X = row_df.loc[:, spec.features].values
        if spec.kind == "trend":
            # многоклассовая классификация
            proba_vec = clf.predict_proba(X)[0]
            classes = spec.classes if spec.classes else list(getattr(clf, "classes_", []))
            # на всякий случай выровняем длины
            if classes and len(classes) == len(proba_vec):
                proba = {str(c): float(p) for c, p in zip(classes, proba_vec)}
                pred = classes[int(np.argmax(proba_vec))]
            else:
                # fallback без имён классов
                proba = {str(i): float(p) for i, p in enumerate(proba_vec)}
                pred = int(np.argmax(proba_vec))
            return {"class": pred, "proba": proba}
        else:
            # риск-модели — бинарные proba[:,1]
            p1 = float(clf.predict_proba(X)[0, 1])
            thr = float(spec.threshold) if spec.threshold is not None else None
            pred = int(p1 >= thr) if thr is not None else None
            return {"proba": p1, "thr": thr, "pred": pred}

    def handle(self, payload: Dict[str, Any]) -> Dict[str, Any]:
        """
        Главная точка входа: валидируем, решаем, что запускать, и возвращаем результат.
        """
        # 1) базовая валидация
        errs = self._validate_basic(payload)
        if errs:
            return {"ok": False, "errors": errs, "ran": [], "missing": {}, "result": {}, "card_id": payload.get("card_id"), "t_sec": payload.get("t_sec")}

        # 2) определяем, что можно запускать
        plan = self._runnable_models(payload)

        # 3) готовим строку признаков
        #    (мы берём как есть весь dict, модели используют только нужные имена из meta)
        feats = payload["features"]
        # приведение типов к float (на всякий случай)
        feats_cast = {k: (float(v) if _numeric(v) else np.nan) for k, v in feats.items()}
        row_df = pd.DataFrame([feats_cast])

        # 4) запускаем что доступно
        result = {}
        ran = []
        missing = {k: v["missing"] for k, v in plan.items() if v["missing"]}
        notes = []
        for name, info in plan.items():
            if info["status"] != "ok":
                if info["reason"]:
                    notes.append(f"{name}: {info['reason']}")
                continue
            spec = self.hub.specs[name]
            try:
                out = self._run_one(name, spec, row_df)
                result[name] = out
                ran.append(name)
            except KeyError as e:
                notes.append(f"{name}: missing feature {str(e)}")
            except Exception as e:
                notes.append(f"{name}: error {type(e).__name__}: {e}")

        return {
            "ok": True,
            "card_id": payload.get("card_id"),  # ИЗМЕНЕНО: было patient_id
            "t_sec": payload.get("t_sec"),
            "ran": ran,
            "missing": missing,          # по моделям → какие поля не хватило
            "notes": notes,              # причины пропусков/ошибок
            "result": result             # предсказания
        }


# ---------- мини-демо ----------
if __name__ == "__main__":
    # Пример: заполните OUT_DIR и payload для ручной проверки
    OUT_DIR = os.getenv("OUT_DIR", r"D:\PythonProjects\LearningToUseP\Checking_ML\out")
    svc = InferenceService(out_dir=OUT_DIR)

    demo_payload = {
        "card_id": "DEMO",  # ИЗМЕНЕНО: было patient_id
        "t_sec": 16*60,   # 16 минут
        "fs_hz": 8,
        "available_windows": ["240s","600s","900s"],
        "features": {
            # сюда положите реальные значения фич; ниже — лишь примеры ключей:
            "f_240s_fhr_mean": 140.0, "f_240s_fhr_std": 5.0, "f_240s_fhr_min": 120.0,
            "f_240s_fhr_max": 160.0, "f_240s_fhr_iqr": 8.0, "f_240s_fhr_rmssd": 2.4,
            "f_240s_fhr_abs_dev": 3.2, "f_240s_fhr_brady_len": 0.0, "f_240s_fhr_tachy_len": 12.0,
            "f_240s_uc_mean": 6.0, "f_240s_uc_std": 2.0, "f_240s_uc_max": 18.0, "f_240s_uc_iqr": 3.0,
            "f_240s_uc_peak_cnt": 1, "f_240s_uc_area": 20.0, "f_240s_fhr_decel_cnt": 0,
            "f_240s_xcorr_maxabs": 0.22, "f_240s_xcorr_lag": 5.0,

            "f_600s_fhr_mean": 141.0, "f_600s_fhr_std": 5.5, "f_600s_fhr_min": 118.0,
            "f_600s_fhr_max": 162.0, "f_600s_fhr_iqr": 9.0, "f_600s_fhr_rmssd": 2.7,
            "f_600s_fhr_abs_dev": 3.3, "f_600s_fhr_brady_len": 0.0, "f_600s_fhr_tachy_len": 30.0,
            "f_600s_uc_mean": 6.5, "f_600s_uc_std": 2.3, "f_600s_uc_max": 19.0, "f_600s_uc_iqr": 3.2,
            "f_600s_uc_peak_cnt": 2, "f_600s_uc_area": 46.0, "f_600s_fhr_decel_cnt": 0,
            "f_600s_xcorr_maxabs": 0.25, "f_600s_xcorr_lag": 6.0,

            "f_900s_fhr_mean": 142.0, "f_900s_fhr_std": 6.2, "f_900s_fhr_min": 116.0,
            "f_900s_fhr_max": 164.0, "f_900s_fhr_iqr": 9.5, "f_900s_fhr_rmssd": 2.9,
            "f_900s_fhr_abs_dev": 3.6, "f_900s_fhr_brady_len": 0.0, "f_900s_fhr_tachy_len": 45.0,
            "f_900s_uc_mean": 7.0, "f_900s_uc_std": 2.5, "f_900s_uc_max": 20.0, "f_900s_uc_iqr": 3.4,
            "f_900s_uc_peak_cnt": 3, "f_900s_uc_area": 70.0, "f_900s_fhr_decel_cnt": 0,
            "f_900s_xcorr_maxabs": 0.28, "f_900s_xcorr_lag": 7.0
        }
    }

    resp = svc.handle(demo_payload)
    print(json.dumps(resp, ensure_ascii=False, indent=2))
