IMAGE_TAG := ${REGION}-docker.pkg.dev/${PROJECT_ID}/${REPO_NAME}/${APP_NAME}

build:
	docker build --platform linux/amd64 -t ${IMAGE_TAG} ..

push: build
	gcloud auth configure-docker ${IMAGE_TAG} --quiet
	docker push ${IMAGE_TAG}

# This will fail firs time because we don't yet have APP_URL.
# Set this once cloud run has created and give us it.
deploy: push
	gcloud run deploy ${APP_NAME} \
	  --image ${IMAGE_TAG} \
	  --region ${REGION} \
	  --project=${PROJECT_ID} \
	  --allow-unauthenticated \
	  --set-env-vars REDIS_URL=${REDIS_URL},GIN_MODE=${GIN_MODE},INSTAGRAM_APP_ID=${INSTAGRAM_APP_ID},INSTAGRAM_SECRET=${INSTAGRAM_SECRET},APP_URL=${APP_URL},API_KEY=${API_KEY}

run: build
	docker run -p 8080:8080 \
	  -e REDIS_URL=${REDIS_URL} \
	  -e GIN_MODE=release \
	  -e INSTAGRAM_APP_ID=${INSTAGRAM_APP_ID} \
	  -e INSTAGRAM_SECRET=${INSTAGRAM_SECRET} \
	  -e APP_URL=${INSTAGRAM_APP_ID} \
	  ${IMAGE_TAG}

