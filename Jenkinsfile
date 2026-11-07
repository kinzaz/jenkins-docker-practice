pipeline {
    agent any

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
    }

    post {
        always {
            archiveArtifacts artifacts: 'report.txt', allowEmptyArchive: true
        }
    }
}
