# Graft

**Graft**는 폐쇄망(Air-gapped) 및 오프라인 배포 환경에 특화된 경량 도커 레지스트리 작업 도구입니다. 리눅스 환경이나 도커 데몬 없이도 이미지 빌드, 수정 및 효율적인 차분 전송(Differential Transfer)을 지원합니다.

## 핵심 특징 (Key Features)

- **오프라인 및 멀티 플랫폼 최적화:** WSL2 설정이 까다로운 오프라인 Windows 환경이나 macOS 환경에서도 별도의 가상화 설정 없이 이미지 빌드와 조작이 가능합니다.
- **경량 데몬리스 빌드 (Daemon-less):** Docker Engine이 설치되지 않은 환경에서도 독립적으로 실행되며, **scratch** 이미지로부터의 빌드를 지원하여 초경량 이미지를 생성할 수 있습니다.
- **차분 추출 (Diff-Pull/Push):** 이미 전체 이미지를 옮길 필요가 없습니다. 변경된 레이어만 추출하여 용량을 최소화하고, 전송 효율을 극대화합니다.
- **직관적인 이미지 수정:** 환경변수 설정, 파일 추가, 엔트리 포인트 변경 등의 작업을 리눅스 환경 없이 즉시 수행합니다.

---

## 주요 기능 (Main Functions)

### 1. Build (Offline & Multi-platform Build)
리눅스 커널이나 WSL2 없이도 `Dockerfile`을 사용해 이미지를 빌드합니다. 오프라인 환경에서의 간단한 이미지 수정 및 배포 준비에 유용합니다.

* **지원하는 Dockerfile 인스트럭션:**
  `FROM`, `COPY`, `ENV`, `WORKDIR`, `ENTRYPOINT`, `EXPOSE`, `CMD`
* **Scratch 빌드 지원:** 베이스 이미지 없이 실행 파일만 포함된 초경량 이미지 생성이 가능합니다.

```bash
# Dockerfile 기반 빌드 예시
graft build -f Dockerfile -t myregistry/myimage:latest --push -u user -p pass
```

### 2. Diff-Pull & Diff-Push (Efficient Transfer)
오프라인 환경으로 이미지를 옮길 때 전송량을 획기적으로 줄여줍니다.

* **Diff-Pull:** 두 이미지 태그 간의 차이점(새로운 레이어)만 추출하여 `.tar` 파일로 저장합니다.
* **Diff-Push:** 추출된 차분 레이어 파일을 대상 레지스트리에 병합합니다.

```bash
# 1. 변경된 레이어만 추출 (외부망)
graft diff-pull --base v1.0 --target v1.1 myregistry/myimage -f diff.tar

# 2. 추출된 파일만 오프라인으로 이동 (USB 등)

# 3. 변경분만 푸시 (내부망)
graft diff-push diff.tar myregistry/myimage:v1.1
```

### 3. Pull & Push
표준 도커 이미지의 Pull/Push 및 `.tar` 파일 내보내기/불러오기를 지원합니다.

---

## 활용 사례 (Use Cases)

### 1. 폐쇄망 Windows/macOS 개발 환경
WSL2나 Docker Desktop 설치가 제한적인 보안 환경에서 Go나 Rust 등으로 빌드된 바이너리를 즉시 컨테이너화하여 내부 레지스트리에 배포할 수 있습니다.

### 2. 대용량 이미지의 효율적 업데이트
이미 운영 중인 서버에 업데이트가 필요할 때, 수 GB에 달하는 전체 이미지를 다시 옮길 필요 없이 `diff-pull`로 변경된 레이어만 빠르게 전송합니다.

---

## 요구 사항 (Requirements)

- **Go 1.26.1** 이상
- **No Docker Daemon Required:** 도커 엔진이나 가상화 레이어 없이 단독 실행 가능

## 의존성 (Dependencies)

- [google/go-containerregistry](https://github.com/google/go-containerregistry): 레지스트리 조작 핵심 라이브러리
- [spf13/cobra](https://github.com/spf13/cobra): CLI 인터페이스 구현
