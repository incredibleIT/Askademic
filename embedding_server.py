#!/usr/bin/env python3
# =============================================================
# WeKnora 本地嵌入服务（fastembed + FastAPI）
#
# 作用: 为 bge-small-zh-v1.5 模型提供 OpenAI 兼容的 /v1/embeddings 接口
#       WeKnora 模型配置: base_url = http://127.0.0.1:8791/v1, provider=generic
#       制度文库知识库的 512 维向量依赖本服务（检索时算查询向量）
#
# 启动: ./start-embedding-server.sh start   （不要直接跑本文件）
# 注意: 首次启动会从 HuggingFace 镜像(hf-mirror.com)下载模型 ~90MB
#       向量行为与旧服务保持一致: 不加任何 query 指令前缀（与已入库向量兼容）
# =============================================================
import os
from typing import List, Optional, Union

from fastapi import FastAPI
from pydantic import BaseModel

MODEL = os.environ.get("EMBED_MODEL", "BAAI/bge-small-zh-v1.5")
PORT = int(os.environ.get("EMBED_PORT", "8791"))
HOST = os.environ.get("EMBED_HOST", "127.0.0.1")

app = FastAPI(title="WeKnora local embedding server")
_model = None
_dim = None


def get_model():
    global _model, _dim
    if _model is None:
        from fastembed import TextEmbedding
        # 本地模型目录优先（环境变量 EMBED_LOCAL_MODEL 指向含 model_optimized.onnx
        # 的目录）：离线加载，跳过 fastembed 的远端校验（hf-mirror 限流时缓存
        # 校验失败会导致整个服务起不来）。缺省仍走 fastembed 缓存/下载。
        local = os.environ.get("EMBED_LOCAL_MODEL", "")
        if local and os.path.isdir(local):
            print(f"[embedding-server] loading local model from {local}", flush=True)
            _model = TextEmbedding(model_name=MODEL, local_model_files=local)
        else:
            _model = TextEmbedding(model_name=MODEL)
        _dim = len(next(iter(_model.embed(["probe"]))))
        print(f"[embedding-server] model ready: {MODEL} dim={_dim}", flush=True)
    return _model


class EmbeddingRequest(BaseModel):
    model: Optional[str] = None
    input: Union[str, List[str]]
    encoding_format: Optional[str] = "float"
    dimensions: Optional[int] = None  # bge 固定 512 维, 忽略该参数


@app.get("/health")
def health():
    return {"status": "ok", "model": MODEL}


@app.get("/v1/models")
def list_models():
    return {
        "object": "list",
        "data": [{"id": "bge-small-zh-v1.5", "object": "model", "owned_by": "local-fastembed"}],
    }


@app.post("/v1/embeddings")
def embeddings(req: EmbeddingRequest):
    texts = [req.input] if isinstance(req.input, str) else [t for t in req.input if t]
    if not texts:
        return {"error": {"message": "input is empty", "type": "invalid_request_error"}}
    m = get_model()
    vectors = [v.tolist() for v in m.embed(texts)]
    return {
        "object": "list",
        "data": [
            {"object": "embedding", "index": i, "embedding": vec}
            for i, vec in enumerate(vectors)
        ],
        "model": req.model or "bge-small-zh-v1.5",
        "usage": {
            "prompt_tokens": sum(len(t) for t in texts),
            "total_tokens": sum(len(t) for t in texts),
        },
    }


if __name__ == "__main__":
    import uvicorn
    get_model()  # 启动时先加载模型, 避免首个请求超时
    uvicorn.run(app, host=HOST, port=PORT, log_level="warning")
