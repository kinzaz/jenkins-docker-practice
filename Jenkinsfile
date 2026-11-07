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

        stage('Check deployment access') {
            steps {
                sh '''
                    ssh \
                        -i /home/jenkins-agent/.ssh/todo_deploy \
                        -o IdentitiesOnly=yes \
                        -o BatchMode=yes \
                        -o StrictHostKeyChecking=yes \
                        -o UserKnownHostsFile=/home/jenkins-agent/.ssh/todo_known_hosts \
                        devops@177.203.59.227 \
                        'whoami && hostname && test -d /home/devops/docker-jenkins'
                '''
            }
        }

        stage('Deliver frontend image') {
            steps {
                sh '''
                    docker save \
                        -o "todo-web-jenkins-$BUILD_NUMBER.tar" \
                        "todo-web:jenkins-$BUILD_NUMBER"

                    scp \
                        -i /home/jenkins-agent/.ssh/todo_deploy \
                        -o IdentitiesOnly=yes \
                        -o BatchMode=yes \
                        -o StrictHostKeyChecking=yes \
                        -o UserKnownHostsFile=/home/jenkins-agent/.ssh/todo_known_hosts \
                        "todo-web-jenkins-$BUILD_NUMBER.tar" \
                        devops@177.203.59.227:/home/devops/docker-jenkins/

                    rm "todo-web-jenkins-$BUILD_NUMBER.tar"
                '''
            }
        }

        stage('Load frontend image') {
            steps {
                sh '''
                    ssh \
                        -i /home/jenkins-agent/.ssh/todo_deploy \
                        -o IdentitiesOnly=yes \
                        -o BatchMode=yes \
                        -o StrictHostKeyChecking=yes \
                        -o UserKnownHostsFile=/home/jenkins-agent/.ssh/todo_known_hosts \
                        devops@177.203.59.227 \
                        "sudo -n /usr/bin/docker load < /home/devops/docker-jenkins/todo-web-jenkins-$BUILD_NUMBER.tar"
                '''
            }
        }

        stage('Activate frontend') {
            steps {
                sh '''
                    ssh \
                        -i /home/jenkins-agent/.ssh/todo_deploy \
                        -o IdentitiesOnly=yes \
                        -o BatchMode=yes \
                        -o StrictHostKeyChecking=yes \
                        -o UserKnownHostsFile=/home/jenkins-agent/.ssh/todo_known_hosts \
                        devops@177.203.59.227 \
                        "/home/devops/docker-jenkins/deploy-web.sh jenkins-$BUILD_NUMBER"
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
