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
    }

    post {
        always {
            archiveArtifacts artifacts: 'report.txt', allowEmptyArchive: true
        }
    }
}
