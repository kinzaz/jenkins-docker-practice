pipeline {
    agent {
        label 'docker-builder'
    }

    parameters {
        booleanParam(
            name: 'SIMULATE_FAILURE',
            defaultValue: false,
            description: 'Fail after activation to test automatic rollback'
        )
        booleanParam(
            name: 'SIMULATE_BACKEND_FAILURE',
            defaultValue: false,
            description: 'Fail after backend activation to test rollback'
        )
    }

    options {
        disableConcurrentBuilds()
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
                        --build-arg "RELEASE=jenkins-$BUILD_NUMBER" \
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

        stage('Check deployment access') {
            steps {
                withCredentials([
                    sshUserPrivateKey(
                        credentialsId: 'todo-deploy-ssh',
                        keyFileVariable: 'DEPLOY_KEY',
                        usernameVariable: 'DEPLOY_USER'
                    )
                ]) {
                    sh '''
                        ssh \
                            -i "$DEPLOY_KEY" \
                            -o IdentitiesOnly=yes \
                            -o BatchMode=yes \
                            -o StrictHostKeyChecking=yes \
                            -o UserKnownHostsFile=/home/jenkins-agent/.ssh/todo_known_hosts \
                            "$DEPLOY_USER@177.203.59.227" \
                            'whoami && hostname && test -d /home/devops/docker-jenkins'
                    '''
                }
            }
        }

        stage('Build backend image') {
            steps {
                sh '''
                    docker buildx build \
                        --platform linux/amd64 \
                        --load \
                        --build-arg "RELEASE=jenkins-$BUILD_NUMBER" \
                        -t "todo-api:jenkins-$BUILD_NUMBER" \
                        backend

                    docker image inspect "todo-api:jenkins-$BUILD_NUMBER" \
                        --format '{{.Os}}/{{.Architecture}} {{.Id}}'
                '''
            }
        }

        stage('Check backend image') {
            steps {
                sh '''
                    set -eu

                    export COMPOSE_PROJECT_NAME="todo-api-ci-$BUILD_NUMBER"

                    cleanup() {
                        docker compose -f compose.ci.yaml down -v
                    }

                    trap cleanup EXIT

                    docker compose -f compose.ci.yaml up -d --wait db

                    docker compose -f compose.ci.yaml exec -T db \
                        psql -v ON_ERROR_STOP=1 -U todo -d todo \
                        < backend/migrations/001_create_todos.sql

                    docker compose -f compose.ci.yaml exec -T db \
                        psql -v ON_ERROR_STOP=1 -U todo -d todo \
                        < backend/migrations/002_add_created_at.sql

                    docker compose -f compose.ci.yaml up -d api

                    docker compose -f compose.ci.yaml run --rm check \
                        --fail --silent --show-error \
                        --retry 15 --retry-delay 2 --retry-all-errors \
                        --connect-timeout 2 --max-time 5 \
                        http://api:8000/api/health

                    docker compose -f compose.ci.yaml run --rm check \
                        --fail --silent --show-error \
                        --retry 15 --retry-delay 2 --retry-all-errors \
                        --connect-timeout 2 --max-time 5 \
                        http://api:8000/api/version \
                        > backend-version.json

                    cat backend-version.json

                    expected=$(printf '{"version":"jenkins-%s"}' "$BUILD_NUMBER")
                    actual=$(cat backend-version.json)

                    test "$actual" = "$expected"
                '''
            }
        }

        stage('Deliver backend image') {
            steps {
                withCredentials([
                    sshUserPrivateKey(
                        credentialsId: 'todo-deploy-ssh',
                        keyFileVariable: 'DEPLOY_KEY',
                        usernameVariable: 'DEPLOY_USER'
                    )
                ]) {
                    sh '''
                        archive="todo-api-jenkins-$BUILD_NUMBER.tar"

                        docker save \
                            -o "$archive" \
                            "todo-api:jenkins-$BUILD_NUMBER"

                        scp \
                            -i "$DEPLOY_KEY" \
                            -o IdentitiesOnly=yes \
                            -o BatchMode=yes \
                            -o StrictHostKeyChecking=yes \
                            -o UserKnownHostsFile=/home/jenkins-agent/.ssh/todo_known_hosts \
                            "$archive" \
                            "$DEPLOY_USER@177.203.59.227:/home/devops/docker-jenkins/"

                        ssh \
                            -i "$DEPLOY_KEY" \
                            -o IdentitiesOnly=yes \
                            -o BatchMode=yes \
                            -o StrictHostKeyChecking=yes \
                            -o UserKnownHostsFile=/home/jenkins-agent/.ssh/todo_known_hosts \
                            "$DEPLOY_USER@177.203.59.227" \
                            "sudo -n /usr/bin/docker load < /home/devops/docker-jenkins/$archive"

                        rm "$archive"
                    '''
                }
            }
        }

        stage('Deliver backend deployment script') {
            steps {
                withCredentials([
                    sshUserPrivateKey(
                        credentialsId: 'todo-deploy-ssh',
                        keyFileVariable: 'DEPLOY_KEY',
                        usernameVariable: 'DEPLOY_USER'
                    )
                ]) {
                    sh '''
                        bash -n scripts/deploy-api.sh

                        scp \
                            -i "$DEPLOY_KEY" \
                            -o IdentitiesOnly=yes \
                            -o BatchMode=yes \
                            -o StrictHostKeyChecking=yes \
                            -o UserKnownHostsFile=/home/jenkins-agent/.ssh/todo_known_hosts \
                            scripts/deploy-api.sh \
                            "$DEPLOY_USER@177.203.59.227:/home/devops/docker-jenkins/deploy-api.sh.next"

                        ssh \
                            -i "$DEPLOY_KEY" \
                            -o IdentitiesOnly=yes \
                            -o BatchMode=yes \
                            -o StrictHostKeyChecking=yes \
                            -o UserKnownHostsFile=/home/jenkins-agent/.ssh/todo_known_hosts \
                            "$DEPLOY_USER@177.203.59.227" \
                            'cd /home/devops/docker-jenkins &&
                             bash -n deploy-api.sh.next &&
                             chmod 700 deploy-api.sh.next &&
                             mv deploy-api.sh.next deploy-api.sh'
                    '''
                }
            }
        }

        stage('Activate backend') {
            steps {
                withCredentials([
                    sshUserPrivateKey(
                        credentialsId: 'todo-deploy-ssh',
                        keyFileVariable: 'DEPLOY_KEY',
                        usernameVariable: 'DEPLOY_USER'
                    )
                ]) {
                    sh '''
                        ssh \
                            -i "$DEPLOY_KEY" \
                            -o IdentitiesOnly=yes \
                            -o BatchMode=yes \
                            -o StrictHostKeyChecking=yes \
                            -o UserKnownHostsFile=/home/jenkins-agent/.ssh/todo_known_hosts \
                            "$DEPLOY_USER@177.203.59.227" \
                            "/home/devops/docker-jenkins/deploy-api.sh jenkins-$BUILD_NUMBER $SIMULATE_BACKEND_FAILURE"
                    '''
                }
            }
        }

        stage('Deliver frontend image') {
            steps {
                withCredentials([
                    sshUserPrivateKey(
                        credentialsId: 'todo-deploy-ssh',
                        keyFileVariable: 'DEPLOY_KEY',
                        usernameVariable: 'DEPLOY_USER'
                    )
                ]) {
                    sh '''
                        docker save \
                            -o "todo-web-jenkins-$BUILD_NUMBER.tar" \
                            "todo-web:jenkins-$BUILD_NUMBER"

                        scp \
                            -i "$DEPLOY_KEY" \
                            -o IdentitiesOnly=yes \
                            -o BatchMode=yes \
                            -o StrictHostKeyChecking=yes \
                            -o UserKnownHostsFile=/home/jenkins-agent/.ssh/todo_known_hosts \
                            "todo-web-jenkins-$BUILD_NUMBER.tar" \
                            "$DEPLOY_USER@177.203.59.227:/home/devops/docker-jenkins/"

                        rm "todo-web-jenkins-$BUILD_NUMBER.tar"
                    '''
                }
            }
        }

        stage('Load frontend image') {
            steps {
                withCredentials([
                    sshUserPrivateKey(
                        credentialsId: 'todo-deploy-ssh',
                        keyFileVariable: 'DEPLOY_KEY',
                        usernameVariable: 'DEPLOY_USER'
                    )
                ]) {
                    sh '''
                        ssh \
                            -i "$DEPLOY_KEY" \
                            -o IdentitiesOnly=yes \
                            -o BatchMode=yes \
                            -o StrictHostKeyChecking=yes \
                            -o UserKnownHostsFile=/home/jenkins-agent/.ssh/todo_known_hosts \
                            "$DEPLOY_USER@177.203.59.227" \
                            "sudo -n /usr/bin/docker load < /home/devops/docker-jenkins/todo-web-jenkins-$BUILD_NUMBER.tar"
                    '''
                }
            }
        }

        stage('Deliver deployment script') {
            steps {
                withCredentials([
                    sshUserPrivateKey(
                        credentialsId: 'todo-deploy-ssh',
                        keyFileVariable: 'DEPLOY_KEY',
                        usernameVariable: 'DEPLOY_USER'
                    )
                ]) {
                    sh '''
                        bash -n scripts/deploy-web.sh

                        scp \
                            -i "$DEPLOY_KEY" \
                            -o IdentitiesOnly=yes \
                            -o BatchMode=yes \
                            -o StrictHostKeyChecking=yes \
                            -o UserKnownHostsFile=/home/jenkins-agent/.ssh/todo_known_hosts \
                            scripts/deploy-web.sh \
                            "$DEPLOY_USER@177.203.59.227:/home/devops/docker-jenkins/deploy-web.sh.next"

                        ssh \
                            -i "$DEPLOY_KEY" \
                            -o IdentitiesOnly=yes \
                            -o BatchMode=yes \
                            -o StrictHostKeyChecking=yes \
                            -o UserKnownHostsFile=/home/jenkins-agent/.ssh/todo_known_hosts \
                            "$DEPLOY_USER@177.203.59.227" \
                            'cd /home/devops/docker-jenkins &&
                             bash -n deploy-web.sh.next &&
                             chmod 700 deploy-web.sh.next &&
                             mv deploy-web.sh.next deploy-web.sh'
                    '''
                }
            }
        }

        stage('Activate frontend') {
            steps {
                withCredentials([
                    sshUserPrivateKey(
                        credentialsId: 'todo-deploy-ssh',
                        keyFileVariable: 'DEPLOY_KEY',
                        usernameVariable: 'DEPLOY_USER'
                    )
                ]) {
                    sh '''
                        ssh \
                            -i "$DEPLOY_KEY" \
                            -o IdentitiesOnly=yes \
                            -o BatchMode=yes \
                            -o StrictHostKeyChecking=yes \
                            -o UserKnownHostsFile=/home/jenkins-agent/.ssh/todo_known_hosts \
                            "$DEPLOY_USER@177.203.59.227" \
                            "/home/devops/docker-jenkins/deploy-web.sh jenkins-$BUILD_NUMBER $SIMULATE_FAILURE"
                    '''
                }
            }
        }
    }

    post {
        always {
            archiveArtifacts artifacts: 'report.txt', allowEmptyArchive: true
        }
    }
}
