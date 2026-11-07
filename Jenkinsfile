pipeline {
    agent {
        label 'docker-builder'
    }

    stages {
        stage('Inspect environment') {
            steps {
                sh '''
                    whoami
                    pwd
                    git rev-parse HEAD
                    test -f backend/go.mod
                    test -f frontend/package-lock.json
                    echo "Report for build $BUILD_NUMBER" > report.txt
                '''
            }
        }

        stage('Check Docker') {
            steps {
                sh '''
                    id
                    docker version
                    docker buildx version
                '''
            }
        }

        stage('Build frontend image') {
            steps {
                sh '''
                    docker buildx build \
                        --platform linux/amd64 \
                        --load \
                        -t "todo-web:jenkins-$BUILD_NUMBER" \
                        frontend

                    docker image inspect "todo-web:jenkins-$BUILD_NUMBER" \
                        --format '{{.Os}}/{{.Architecture}} {{.Id}}'
                '''
            }
        }

        stage('Check frontend image') {
            steps {
                sh '''
                    docker run --rm \
                        --entrypoint sh \
                        "todo-web:jenkins-$BUILD_NUMBER" \
                        -ec '
                            test -s /usr/share/nginx/html/index.html
                            test -d /usr/share/nginx/html/assets
                            nginx -t
                        '
                '''
            }
        }
    }

    post {
        always {
            archiveArtifacts artifacts: 'report.txt', allowEmptyArchive: true
        }
    }
}
